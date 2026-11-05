package main

import (
	"math"
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

// Câu cá ở ao (PLAN_LOGIC_GAME 16): đứng ở đầu cầu ao, chờ phao chìm, giằng co như đào giun; cá vào kho và ăn
// được; cá to giật đứt mồi thì không có cá nhưng vẫn tính hồi câu.
func TestFishingAtPond(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "cauca", "Gà Câu Cá", func(acc *Account) { acc.Mana = 50 })
	e.warp(a, "park")
	if r := a.act("fish_start", msg{"id": "cau_ca"}); r["ok"] == true {
		t.Fatal("phải ra đầu cầu ao mới câu được")
	}
	a.walkToItem("park", "cau_ca")
	e.forceRoll(t, 0.1) // cá rô
	start := a.mustOK("fish_start", msg{"id": "cau_ca"})
	d := start["data"].(map[string]any)
	fish := d["fish"].(map[string]any)["id"].(string)
	if fish != "ca_ro" || d["biteMs"].(float64) <= 0 {
		t.Fatalf("thả câu: %v", d)
	}
	if r := a.act("fish_finish", a.tugPlan(start, false)); r["ok"] == true {
		t.Fatal("kéo lên trước khi phao chìm phải bị từ chối")
	}
	res := a.mustOK("fish_finish", a.tugPlan(start, true))["data"].(map[string]any)
	if res["caught"] != true || a.count("items", fish) != 1 || a.num("fishCaught") != 1 {
		t.Fatalf("câu được cá vào kho: %v", res)
	}
	if r := a.act("fish_start", msg{"id": "cau_ca"}); r["ok"] == true {
		t.Fatal("cá còn lặn: phải chờ hồi câu")
	}
	mana := a.num("mana")
	a.mustOK("eat_item", msg{"id": fish})
	if a.num("mana") <= mana || a.count("items", fish) != 0 {
		t.Fatal("ăn cá hồi độ no")
	}
	// Cá trê (Hiếm): không giữ nút thì hết giờ cá sổng.
	time.Sleep(time.Duration(a.num("fishReadyIn")*float64(time.Second)) + 100*time.Millisecond)
	e.forceRoll(t, 0.95)
	start = a.mustOK("fish_start", msg{"id": "cau_ca"})
	d = start["data"].(map[string]any)
	if d["tier"].(float64) < 2 {
		t.Fatalf("cần cá bậc Hiếm: %v", d["fish"])
	}
	p := ScaleTug(TugTiers[int(d["tier"].(float64))], e.hub.bal.TugSpeed)
	time.Sleep(time.Duration(d["biteMs"].(float64)+d["revealMs"].(float64))*time.Millisecond + time.Duration(p.MaxSec*float64(time.Second)) + 50*time.Millisecond)
	miss := a.mustOK("fish_finish", msg{"id": d["id"], "ev": [][2]float64{}})["data"].(map[string]any)
	if miss["escaped"] != true || a.count("items", "ca_tre") != 0 || a.num("fishReadyIn") <= 0 {
		t.Fatalf("cá sổng vẫn tính hồi câu: %v", miss)
	}
}

func TestRollFishTable(t *testing.T) {
	sum := 0.0
	for _, f := range FishTable {
		sum += f.Prob
		if it := itemByID(f.Item); it == nil || it.Sell <= 0 || it.Mana <= 0 {
			t.Fatalf("cá %s phải bán và ăn được", f.Item)
		}
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("bảng cá cộng %.4f", sum)
	}
	for r := 0.0; r < 1; r += 0.01 {
		if RollFish(r, true, true).Tier > 1 {
			t.Fatal("kiệt sức chỉ câu được cá bậc thấp")
		}
	}
	if RollFish(0.999, false, false).Item != "ca_vang" || RollFish(0.975, false, true).Item != "ca_vang" || RollFish(0.975, false, false).Item == "ca_vang" {
		t.Fatal("đêm cá vàng ăn mồi nhiều hơn")
	}
}

// Tưới giúp luống của bạn bè (PLAN_LOGIC_GAME 14): chỉ bạn bè, đứng cạnh luống, mỗi cây một lần tưới; người
// tưới được +1 thân thiết cho bạn đồng hành, người trồng được báo và vẫn là người duy nhất thu hoạch.
func TestWaterFriendsBed(t *testing.T) {
	e := newEnv(t, t.TempDir())
	setup := func(acc *Account) {
		acc.Companions = []Companion{{ID: "c1", Name: "Bé", Species: "ga_con"}}
		acc.Following = "c1"
		acc.Items["hat_bi_ngo"] = 1 // bộ test: cây lớn nhanh ×1000, bí ngô chín sau 2,7 giây
	}
	a, aid := e.seeded(t, "trongcay", "Gà Trồng", setup)
	b, bid := e.seeded(t, "tuoigiup", "Gà Tưới", setup)
	e.warp(a, "park")
	e.warp(b, "park")
	a.walkToItem("park", "luong_0")
	b.walkToItem("park", "luong_0")
	a.mustOK("plant", msg{"id": "luong_0", "mode": "hat_bi_ngo"})
	apid := e.pidOf(a)
	if r := b.act("water_friend", msg{"id": "luong_0", "target": apid}); r["ok"] == true {
		t.Fatal("chưa là bạn thì không tưới giúp")
	}
	if _, err := e.store.Social.Request(aid, bid); err != nil {
		t.Fatal(err)
	}
	if err := e.store.Social.Respond(bid, aid, true); err != nil {
		t.Fatal(err)
	}
	r := b.mustOK("water_friend", msg{"id": "luong_0", "target": apid})
	if c := r["me"].(map[string]any)["companions"].([]any)[0].(map[string]any); c["affinity"].(float64) != 1 {
		t.Fatalf("người tưới: bạn đồng hành +1 thân thiết: %v", c)
	}
	a.toastHas("tưới giúp")
	fb := b.expectWhere("fbeds", 3*time.Second, func(m msg) bool { return len(m["beds"].([]any)) > 0 })
	if bed := fb["beds"].([]any)[0].(map[string]any); bed["key"] != "park:luong_0" || bed["watered"] != true || bed["pid"] != apid {
		t.Fatalf("bạn bè thấy cây đã tưới: %v", bed)
	}
	beds := a.mustOK("title", msg{"id": ""})["me"].(map[string]any)["beds"].(map[string]any)
	if beds["park:luong_0"].(map[string]any)["watered"] != true {
		t.Fatal("cây của người trồng phải được tưới")
	}
	if r := b.act("water_friend", msg{"id": "luong_0", "target": apid}); r["ok"] == true {
		t.Fatal("mỗi cây tưới một lần")
	}
	if r := b.act("harvest", msg{"id": "luong_0"}); r["ok"] == true {
		t.Fatal("người tưới giúp không thu hoạch cây của bạn")
	}
}

// Lô trưng bày ở Xóm Rơm Vàng (PLAN_LOGIC_GAME 13.2, P5b): mỗi người thấy nhà mình trước, rồi nhà bạn bè, rồi
// nhà người khác đang online trong kênh; lô còn lại là nhà mẫu. Từ cổng lô vào thăm được nhà bạn.
func TestShowcaseLotsInNeighborhood(t *testing.T) {
	e := newEnv(t, t.TempDir())
	build := func(id, name, design, material, state string) {
		h, _ := e.store.Homes.Get(id, name)
		h.Tier = 2
		h.Coop = &CoopJob{Roof: "xanh", Design: design, Material: material, State: state, DoneAt: time.Now().Add(time.Hour)}
		e.store.Homes.Save(h)
	}
	a, aid := e.seeded(t, "chunha1", "Gà Nhà Gạch", func(*Account) {})
	b, bid := e.seeded(t, "chunha2", "Gà Nhà Gỗ", func(*Account) {})
	c, _ := e.seeded(t, "khongnha", "Gà Chưa Nhà", func(*Account) {})
	build(aid, "Gà Nhà Gạch", "nha_hai_gian", "gach", "done")
	build(bid, "Gà Nhà Gỗ", "chuong_nho", "go", "building")
	e.store.Social.Request(aid, bid)
	e.store.Social.Respond(bid, aid, true)
	for _, x := range []*client{a, b, c} {
		e.warp(x, "neighborhood")
	}
	lots := func(x *client) []any {
		return x.expectWhere("showcase", 8*time.Second, func(m msg) bool {
			n := 0
			for _, l := range m["lots"].([]any) {
				if l != nil {
					n++
				}
			}
			return n == 2
		})["lots"].([]any)
	}
	la := lots(a)
	if len(la) != len(ShowcaseLots) || la[0].(map[string]any)["owner"] != aid || la[0].(map[string]any)["mine"] != true ||
		la[1].(map[string]any)["owner"] != bid || la[1].(map[string]any)["friend"] != true || la[2] != nil {
		t.Fatalf("A thấy nhà mình rồi tới nhà bạn, còn lại nhà mẫu: %v", la)
	}
	if l := la[1].(map[string]any); l["design"] != "chuong_nho" || l["material"] != "go" || l["state"] != "building" {
		t.Fatalf("lô của bạn vẽ đúng nhà đang xây: %v", l)
	}
	if lc := lots(c); lc[0].(map[string]any)["owner"] != aid || lc[1].(map[string]any)["owner"] != bid {
		t.Fatalf("người chưa có nhà thấy nhà người khác theo thứ tự vào kênh: %v", lc)
	}
	// Từ cổng lô của bạn vào thăm nhà bạn.
	a.walkToItem("neighborhood", "lo_2")
	a.send(msg{"t": "visit", "owner": bid})
	if w := a.expect("welcome", 3*time.Second); w["home"].(map[string]any)["owner"] != bid {
		t.Fatal("vào thăm nhà bạn từ lô trưng bày")
	}
}
