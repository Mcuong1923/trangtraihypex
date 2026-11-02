package main

import (
	"math/rand/v2"
	"testing"
	"time"
)

// E2E của PLAN_LOGIC_GAME 21: mua đất và xây nhà; 0 thóc 0 độ no vẫn chơi tiếp được; giun hiếm thoát
// không tính vào hạn đào trong ngày.

// queuedSource trả lần lượt các số định sẵn rồi quay về nguồn thường: ép máy chủ gieo đúng loài giun cần
// thử mà không đụng vào luật đào.
type queuedSource struct {
	q    []uint64
	next rand.Source
}

func (s *queuedSource) Uint64() uint64 {
	if len(s.q) > 0 {
		v := s.q[0]
		s.q = s.q[1:]
		return v
	}
	return s.next.Uint64()
}

// forceRoll cho lượt rút số kế tiếp của phòng ra giá trị r (Float64 của rand/v2 dùng 53 bit thấp). Xếp sẵn
// nhiều số giống nhau phòng khi nhịp phòng rút trước vài số cho NPC giao thông.
func (e *env) forceRoll(t *testing.T, r float64) {
	room := e.roomOf(t)
	q := make([]uint64, 40)
	for i := range q {
		q[i] = uint64(r * (1 << 53))
	}
	room.DoWait(func() { room.rng = rand.New(&queuedSource{q: q, next: rand.NewPCG(1, 2)}) })
}

func (c *client) field(key string) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.me[key]
}

func TestBuyLotAndBuildHouseE2E(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, id := e.seeded(t, "chudat", "Gà Mua Đất", func(acc *Account) { acc.Level = 3; acc.Thoc = 3000 })
	e.warp(a, "neighborhood")
	if r := a.act("land_buy", msg{"i": 1, "mode": "ven_ao"}); r["ok"] == true {
		t.Fatal("mua đất phải gặp Ông Chủ Đầu Tư")
	}
	a.walkToItem("neighborhood", "chu_dau_tu")
	info := a.mustOK("land_info", nil)["data"].(map[string]any)
	if info["tier"].(float64) != 0 || info["place"] != "duong" || len(info["places"].([]any)) != 3 {
		t.Fatalf("bảng lô đất: %v", info)
	}
	if r := a.act("land_buy", msg{"i": 2, "mode": "duong"}); r["ok"] == true {
		t.Fatal("Lô rộng cần cấp 6")
	}
	a.mustOK("land_buy", msg{"i": 1, "mode": "ven_ao"})
	if a.num("thoc") != 3000-1200-300 {
		t.Fatalf("lô tiêu chuẩn 1.200 + phụ phí ven ao 300: còn %v", a.num("thoc"))
	}
	// Tiệm vật liệu của Bò Mộc ở Xóm.
	a.walkToItem("neighborhood", "bo_moc")
	a.mustOK("item_buy", msg{"id": "rom", "i": 20})
	a.mustOK("item_buy", msg{"id": "go", "i": 5})
	if a.num("thoc") != 1500-20*3-5*8 || a.count("items", "rom") != 20 || a.count("items", "go") != 5 {
		t.Fatalf("mua vật liệu: thóc %v rơm %v gỗ %v", a.num("thoc"), a.count("items", "rom"), a.count("items", "go"))
	}

	a.send(msg{"t": "visit", "owner": "me"})
	w := a.expect("welcome", 2*time.Second)
	if h := w["home"].(map[string]any); h["tier"].(float64) != 1 || h["place"] != "ven_ao" {
		t.Fatalf("vào nhà phải thấy lô mới: %v %v", h["tier"], h["place"])
	}
	home, _ := e.store.Homes.Get(id, "Gà Mua Đất")
	cow := home.Zone().Item("bo_moc")
	a.walkTo(cow.X, cow.Z+0.8)
	if r := a.act("home_build", msg{"id": "do", "to": "chuong_nho", "mode": "gach"}); r["ok"] == true || a.num("thoc") != 1400 {
		t.Fatal("thiếu gạch thì không xây, không trừ gì")
	}
	if r := a.act("home_build", msg{"id": "do", "to": "nha_hai_gian", "mode": "go"}); r["ok"] == true {
		t.Fatal("Nhà hai gian cần Lô rộng")
	}
	a.mustOK("home_build", msg{"id": "do", "to": "chuong_nho", "mode": "rom"})
	if a.num("thoc") != 1200 || a.count("items", "rom") != 0 || a.count("items", "go") != 0 {
		t.Fatalf("chuồng rơm: 200 thóc + 20 rơm + 5 gỗ: thóc %v", a.num("thoc"))
	}
	// Bộ test xây 10 phút × 1/600 = 1 giây; phòng tự nghiệm thu rồi gửi nhà mới.
	done := a.expectWhere("home", 5*time.Second, func(m msg) bool {
		c, _ := m["home"].(map[string]any)["coop"].(map[string]any)
		return c != nil && c["state"] == "done"
	})["home"].(map[string]any)
	c := done["coop"].(map[string]any)
	if c["design"] != "chuong_nho" || c["material"] != "rom" || done["regen"].(float64) != 1 || done["cap"].(float64) != 8 {
		t.Fatalf("chuồng rơm xong: hồi +1/phút, chứa 8: %v", done)
	}
	// Nâng rơm → gỗ: chỉ trả phần vật liệu chênh lệch (15 gỗ, 10 đinh) và nửa tiền công (200).
	a.mustOK("item_buy", msg{"id": "go", "i": 15})
	a.mustOK("item_buy", msg{"id": "dinh", "i": 10})
	thoc := a.num("thoc")
	a.mustOK("home_upgrade_house", msg{"mode": "go"})
	if a.num("thoc") != thoc-200 || a.count("items", "go") != 0 || a.count("items", "dinh") != 0 {
		t.Fatalf("nâng cấp gỗ: thóc %v → %v", thoc, a.num("thoc"))
	}
	up := a.expectWhere("home", 5*time.Second, func(m msg) bool {
		c, _ := m["home"].(map[string]any)["coop"].(map[string]any)
		return c != nil && c["state"] == "done" && c["material"] == "go"
	})["home"].(map[string]any)
	if up["regen"].(float64) != 1.5 || up["cap"].(float64) != 12 {
		t.Fatalf("nhà gỗ: hồi +1,5/phút, chứa 12: %v %v", up["regen"], up["cap"])
	}
}

func TestZeroThocZeroManaStillPlayable(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "trangtay", "Gà Trắng Tay", func(acc *Account) {
		acc.Thoc, acc.Mana, acc.ZeroElapsed = 0, 0, 10000
		acc.Worms, acc.Food, acc.Items = map[string]int{}, map[string]int{}, map[string]int{}
	})
	if a.num("stage") != 4 || a.field("rationReady") != true {
		t.Fatalf("bối cảnh: kiệt sức, được xin cứu trợ: stage %v ration %v", a.num("stage"), a.field("rationReady"))
	}
	// Cách 1: Cụ Cục dúi một phần cám mỗi ngày khi trắng tay.
	a.walkToItem("lobby", "ga_cu_cuc")
	a.mustOK("ration", nil)
	if a.count("food", "cam") != 1 {
		t.Fatal("cứu trợ: 1 cám")
	}
	if r := a.act("ration", nil); r["ok"] == true {
		t.Fatal("mỗi ngày một phần")
	}
	// Cách 2: kiệt sức vẫn đào được (chỉ giun thường/khá hiếm), ăn sống ngay tại vườn.
	e.warp(a, "garden")
	a.walkToItem("garden", "dat_a")
	start := a.mustOK("dig_start", msg{"id": "dat_a"})
	d := start["data"].(map[string]any)
	if d["exhausted"] != true || d["tier"].(float64) > 1 {
		t.Fatalf("đào khi kiệt sức chỉ ra giun bậc thấp: %v", d)
	}
	res := a.mustOK("dig_finish", a.tugPlan(start, true))
	data := res["data"].(map[string]any)
	worm, _ := data["loot"].(map[string]any)["worm"].(string)
	if data["caught"] != true || a.count("worms", worm) != 1 {
		t.Fatalf("kiệt sức vẫn kéo được giun về túi: %v", data)
	}
	a.mustOK("eat_raw", msg{"id": worm})
	if a.num("mana") <= 0 || a.count("worms", worm) != 0 {
		t.Fatalf("ăn sống phải hồi độ no: %v", a.num("mana"))
	}
}

func TestRareWormEscapeNotCountedDaily(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "huthoi", "Gà Hụt Giun", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "garden")
	a.walkToItem("garden", "dat_a")
	// r = 0.96: sau phần giun mùa thu (0,5%) rơi vào dải Giun phát sáng (Hiếm).
	e.forceRoll(t, 0.96)
	start := a.mustOK("dig_start", msg{"id": "dat_a"})
	d := start["data"].(map[string]any)
	if d["tier"].(float64) < 2 {
		t.Fatalf("cần giun từ bậc Hiếm: %v", d["worm"])
	}
	daily, mana := a.num("dailyDigs"), a.num("mana")
	// Không giữ nút lần nào: giun hiếm hết giờ thì thoát. Gửi sau đúng thời gian giằng co.
	p := ScaleTug(TugTiers[int(d["tier"].(float64))], e.hub.bal.TugSpeed)
	time.Sleep(time.Duration(d["peckMs"].(float64)+d["revealMs"].(float64))*time.Millisecond + time.Duration(p.MaxSec*float64(time.Second)) + 50*time.Millisecond)
	r := a.mustOK("dig_finish", msg{"id": d["id"], "ev": [][2]float64{}})["data"].(map[string]any)
	if r["escaped"] != true || r["caught"] == true {
		t.Fatalf("giun hiếm phải thoát: %v", r)
	}
	if a.num("dailyDigs") != daily || a.num("xp") < EscapeXP {
		t.Fatalf("giun thoát không tính hạn ngày, vẫn có XP an ủi: lượt %v → %v", daily, a.num("dailyDigs"))
	}
	if a.num("mana") >= mana || a.num("digReadyIn") <= 0 {
		t.Fatal("giun thoát vẫn tốn sức và tính hồi đất (chống gieo lại tìm giun hiếm)")
	}
	// Bỏ dở giữa chừng cũng tính như giun thoát.
	time.Sleep(time.Duration(a.num("digReadyIn")*float64(time.Second)) + 50*time.Millisecond)
	e.forceRoll(t, 0.96)
	a.mustOK("dig_start", msg{"id": "dat_a"})
	a.mustOK("dig_cancel", nil)
	if r := a.act("dig_start", msg{"id": "dat_a"}); r["ok"] == true {
		t.Fatal("bỏ dở rồi đào lại ngay phải chờ hồi đất")
	}
	if a.num("dailyDigs") != daily {
		t.Fatal("bỏ dở không tính hạn ngày")
	}
}
