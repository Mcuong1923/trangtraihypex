package main

import (
	"strings"
	"testing"
	"time"
)

// Test hồi quy cho nhóm "đồng bộ với plan" (lượt rà 2, mục A1–A4, A7, D3 và các câu hỏi plan đã có đáp án)
// trong plans/reports/code-reviewer-261105-1412-game-logic-review.md.

// D3, A7: chỉ Mổ nhanh giữa hai người thật được tính thắng (PLAN.md 10.1: may rủi không dùng cho xếp hạng).
func TestPKRankedOnlyHumanPeck(t *testing.T) {
	cases := []struct {
		m    Match
		want bool
	}{
		{Match{Mode: "peck"}, true},
		{Match{Mode: "coin"}, false},
		{Match{Mode: "peck", Bot: true}, false},
		{Match{Mode: "coin", Bot: true}, false},
	}
	for _, c := range cases {
		if got := c.m.ranked(); got != c.want {
			t.Fatalf("%s bot=%v: ranked %v, muốn %v", c.m.Mode, c.m.Bot, got, c.want)
		}
	}
}

// D3: thắng Gà Máy không cộng số trận thắng (danh hiệu "Cao thủ PK"), nhưng vẫn tính đã chơi.
func TestBotWinNotCountedForTitle(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, id := e.seeded(t, "thangmay", "Gà Thắng Máy", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "arena")
	a.walkToItem("arena", "cho_muc")
	a.mustOK("pk_bot", msg{"mode": "coin"})
	a.expect("pk_start", 2*time.Second)
	// Ép người chơi thắng để test luôn kiểm đúng nhánh "thắng Gà Máy" (úp ngửa vốn ngẫu nhiên).
	room := e.roomOf(t)
	room.DoWait(func() {
		for _, mt := range room.matches {
			mt.ScoreA += 10
		}
	})
	for round := 0; round < CoinRounds; round++ {
		a.expect("pk_round", 5*time.Second)
		e.expireCoinPicks()
		a.expect("pk_flip", 3*time.Second)
	}
	a.expect("pk_end", 6*time.Second)
	acc, _ := e.store.Account(id)
	var played, wins int
	e.roomOf(t).DoWait(func() { played, wins = acc.Stats.PKPlayed, acc.Stats.PKWins })
	if played != 1 || wins != 0 {
		t.Fatalf("đấu Gà Máy: đã chơi %d (muốn 1), thắng tính %d (muốn 0)", played, wins)
	}
}

// A4: gà con nở trước v2 (loài trống) giữ "Gà con" và có kỹ năng "Mỏ Thính" (PLAN_LOGIC_GAME 15.2).
func TestLegacyChickGetsSpecies(t *testing.T) {
	a := newAcc()
	a.Companions = []Companion{{ID: "pet1", Name: "Bé Vàng"}, {ID: "pet2", Species: "vit_con"}}
	a.Following = "pet1"
	a.normalize()
	if a.Companions[0].Species != "ga_con" || a.Companions[1].Species != "vit_con" {
		t.Fatalf("chuyển loài: %q, %q", a.Companions[0].Species, a.Companions[1].Species)
	}
	if petBonus(a, AbilitySniff) <= 0 {
		t.Fatal("gà con cũ đang đi theo phải có Mỏ Thính")
	}
}

// A2: sức chứa khách theo vật liệu nhà: rơm 8, gỗ 12, gạch 16 (PLAN_LOGIC_GAME 13.3); chưa có nhà thì 8.
func TestHouseCapacityByMaterial(t *testing.T) {
	for mat, want := range map[string]int{"": HomeCapacity, "rom": 8, "go": 12, "gach": 16} {
		if got := houseCapacity(mat, "chuong_nho"); got != want {
			t.Fatalf("vật liệu %q: sức chứa %d, muốn %d", mat, got, want)
		}
	}
	dir := t.TempDir()
	hs, err := OpenHomes(dir)
	if err != nil {
		t.Fatal(err)
	}
	owner := strings.Repeat("ab", AccountIDLen/2)
	h, _ := hs.Get(owner, "Gà Gạch")
	h.Coop = &CoopJob{Design: "chuong_nho", Material: "gach", State: "done"}
	if err := hs.Save(h); err != nil {
		t.Fatal(err)
	}
	if got := hs.Capacity(owner, t0); got != 16 {
		t.Fatalf("nhà gạch qua chỉ mục: %d khách, muốn 16", got)
	}
	// Đang nâng cấp: chưa tính sức chứa nhà mới; quá DoneAt (chủ offline, State cũ) thì tính.
	h.Coop = &CoopJob{Design: "chuong_nho", Material: "gach", State: "building", DoneAt: t0.Add(time.Hour)}
	_ = hs.Save(h)
	if got := hs.Capacity(owner, t0); got != HomeCapacity {
		t.Fatalf("đang xây: %d khách, muốn %d", got, HomeCapacity)
	}
	if got := hs.Capacity(owner, t0.Add(2*time.Hour)); got != 16 {
		t.Fatalf("quá giờ xây xong: %d khách, muốn 16", got)
	}
	if got := hs.Capacity(strings.Repeat("cd", AccountIDLen/2), t0); got != HomeCapacity {
		t.Fatalf("nhà chưa có: %d khách", got)
	}
}

// Quà lên cấp khi kho đầy vào hàng chờ, không mất (PLAN.md 7.1); có chỗ thì nhận vào kho.
func TestLevelRewardQueuedWhenStorageFull(t *testing.T) {
	a := newAcc()
	a.Items["rom"] = StorageCap // kho đầy
	a.Level = 3                 // cấp 3: 10 rơm, 5 gỗ, 2 hạt cà rốt
	got := a.ClaimLevelRewards(t0)
	if !got[3].Queued || a.PendingItems["go"] != 5 || a.PendingItems["rom"] != 10 {
		t.Fatalf("quà cấp 3 phải vào hàng chờ: %+v, chờ %v", got[3], a.PendingItems)
	}
	if _, err := a.ClaimPending(); err == nil {
		t.Fatal("kho còn đầy thì chưa nhận được")
	}
	a.Items["rom"] = StorageCap - 4 // trống 4 chỗ: nhận một phần, phần còn lại vẫn chờ
	moved, err := a.ClaimPending()
	if err != nil || sum(moved) != 4 || a.StorageCount() != StorageCap || a.pendingCount() != 10+5+2+2-4 {
		t.Fatalf("nhận một phần: %v %v, còn chờ %d", moved, err, a.pendingCount())
	}
	a.Items = map[string]int{}
	if _, err := a.ClaimPending(); err != nil || len(a.PendingItems) != 0 {
		t.Fatalf("kho trống thì nhận hết: %v, còn %v", err, a.PendingItems)
	}
	a.PendingItems = map[string]int{"go": PendingCap - 2}
	if dropped := a.queuePending("da", 5); dropped != 3 || a.pendingCount() != PendingCap {
		t.Fatalf("hàng chờ có giới hạn: bỏ %d, còn %d", dropped, a.pendingCount())
	}
}

func sum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

// A3: giao dịch trao đổi được vật phẩm kho (PLAN_LOGIC_GAME 11), không được trứng; kho người nhận đầy thì huỷ cả hai.
func TestTradeItems(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, aid := e.seeded(t, "tia", "Gà Gỗ", func(acc *Account) { acc.Level = 3; acc.Items["go"] = 10; acc.Items["trung_thuong"] = 1 })
	b, bid := e.seeded(t, "tib", "Gà Thóc", func(acc *Account) { acc.Level = 3; acc.Thoc = 50 })
	a.expect("pjoin", 2*time.Second)
	pos := a.position()
	b.walkTo(pos.X+1, pos.Z)
	accA, _ := e.store.Account(aid)
	accB, _ := e.store.Account(bid)
	var aGo0, bGo0, aThoc0 int
	e.roomOf(t).DoWait(func() { aGo0, bGo0, aThoc0 = accA.Items["go"], accB.Items["go"], accA.Thoc })
	id := openTradeWith(t, e, a, b, "Gà Thóc")
	if r := a.act("trade_offer", msg{"id": id, "items": map[string]int{"trung_thuong": 1}}); r["ok"] == true {
		t.Fatal("trứng không giao dịch được")
	}
	a.mustOK("trade_offer", msg{"id": id, "items": map[string]int{"go": 6}})
	b.mustOK("trade_offer", msg{"id": id, "i": 40})
	a.mustOK("trade_lock", msg{"id": id})
	b.mustOK("trade_lock", msg{"id": id})
	a.mustOK("trade_confirm", msg{"id": id})
	b.mustOK("trade_confirm", msg{"id": id})
	var aGo, bGo, aThoc int
	e.roomOf(t).DoWait(func() { aGo, bGo, aThoc = accA.Items["go"], accB.Items["go"], accA.Thoc })
	if aGo != aGo0-6 || bGo != bGo0+6 || aThoc != aThoc0+40 {
		t.Fatalf("sau giao dịch: A %d gỗ %d thóc, B %d gỗ", aGo, aThoc, bGo)
	}

	// Kho B đầy: giao dịch bị huỷ, không bên nào mất gì.
	e.roomOf(t).DoWait(func() { accB.Items["rom"] += StorageCap - accB.StorageCount() })
	id = openTradeWith(t, e, a, b, "Gà Thóc")
	a.mustOK("trade_offer", msg{"id": id, "items": map[string]int{"go": 4}})
	a.mustOK("trade_lock", msg{"id": id})
	b.mustOK("trade_lock", msg{"id": id})
	a.mustOK("trade_confirm", msg{"id": id})
	if r := b.act("trade_confirm", msg{"id": id}); r["ok"] == true || !strings.Contains(r["err"].(string), "Kho đầy") {
		t.Fatalf("kho người nhận đầy phải huỷ: %v", r)
	}
	e.roomOf(t).DoWait(func() { aGo, bGo = accA.Items["go"], accB.Items["go"] })
	if aGo != aGo0-6 || bGo != bGo0+6 {
		t.Fatalf("huỷ thì không đổi gì: A %d gỗ, B %d gỗ", aGo, bGo)
	}
}

// openTradeWith mời giao dịch người tên name và để người đó nhận lời; trả ID giao dịch.
func openTradeWith(t *testing.T, e *env, from, to *client, name string) any {
	t.Helper()
	var pid string
	room := e.roomOf(t)
	room.DoWait(func() {
		for _, x := range room.players {
			if x.acc.Char.Name == name {
				pid = x.PID
			}
		}
	})
	from.mustOK("trade_invite", msg{"target": pid})
	id := to.expect("trade_invite", 2*time.Second)["id"]
	to.mustOK("trade_accept", msg{"id": id})
	return id
}

// A1: xe liên trại chạy theo lịch cố định; chỉ giữ ghế khi cửa mở (BusWait trước giờ chạy, PLAN_LOGIC_GAME 7.3).
func TestIntercityBookingOnlyWhenDoorOpen(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.BusEvery, b.BusWait = 1000*time.Hour, time.Second })
	a, _ := e.seeded(t, "choxe", "Gà Chờ Xe", func(acc *Account) { acc.Level = 3 })
	a.goZone("ra_ben", "station")
	a.walkToItem("station", "tram_bus")
	r := a.act("book", nil)
	if r["ok"] == true || !strings.Contains(r["err"].(string), "cửa mở") {
		t.Fatalf("ngoài giờ mở cửa không giữ được ghế: %v", r)
	}
	if a.num("tripsLeft") != MaxBusTokens {
		t.Fatal("không giữ được ghế thì không trừ lượt")
	}
}
