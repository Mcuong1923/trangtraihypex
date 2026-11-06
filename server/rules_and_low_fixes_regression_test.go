package main

import (
	"strings"
	"testing"
	"time"
)

// Test cho các luật chốt ngày 06/11/2026 và nhóm lỗi M7–M11 + lỗi thấp phía server
// (plans/reports/code-reviewer-261105-1412-game-logic-review.md).

// Luật 1: luống rau nhà theo lô (Tổ nhỏ 2, tiêu chuẩn 4, rộng 6, đồi 8).
func TestHomeBedLimitByLot(t *testing.T) {
	want := []int{2, 4, 6, 8}
	for i, tr := range LandTiers {
		if tr.Beds != want[i] {
			t.Fatalf("%s: %d luống, muốn %d", tr.Name, tr.Beds, want[i])
		}
	}
	h := NewHome(strings.Repeat("1", AccountIDLen), "Gà Rau")
	h.Storage["luong_rau"] = 3
	for i, x := range []int{-3, 3} {
		if _, err := h.Place("luong_rau", x, -1, 0); err != nil {
			t.Fatalf("luống %d: %v", i+1, err)
		}
	}
	if _, err := h.Place("luong_rau", 0, -2, 0); err == nil || !strings.Contains(err.Error(), "tối đa 2") {
		t.Fatalf("Tổ nhỏ chỉ đặt 2 luống: %v", err)
	}
	a := newAcc()
	a.Beds["home:o1"], a.Beds["park:luong_0"], a.Beds["home:o2"] = &Plant{}, &Plant{}, &Plant{}
	if a.homePlantings() != 2 {
		t.Fatal("chỉ đếm luống nhà")
	}
}

// Luật 2, 3: câu cá 30 con/ngày; thân thiết từ bạn bè tối đa 60/ngày, sang ngày mới đếm lại.
func TestFishCapAndSocialAffinityCap(t *testing.T) {
	if FishDailyDefault != 30 {
		t.Fatalf("trần câu cá %d, đã chốt 30", FishDailyDefault)
	}
	a := newAcc()
	c := &Companion{ID: "pet1", Species: "ga_con"}
	for i := 0; i < 70; i++ {
		a.socialAffinity(c, t0)
	}
	if c.Affinity != SocialAffinityPerDay {
		t.Fatalf("một ngày nhận %d thân thiết từ bạn bè, trần %d", c.Affinity, SocialAffinityPerDay)
	}
	if !a.socialAffinity(c, t0.Add(24*time.Hour)) {
		t.Fatal("sang ngày mới được tính lại")
	}
}

// Luật 4 + chống thông đồng: thắng cùng một đối thủ quá PKPairPerWeek trận/tuần không tính thêm.
func TestPKPairWeeklyCap(t *testing.T) {
	a := newAcc()
	for i := 0; i < PKPairPerWeek; i++ {
		if !a.countPKWin("doi_thu", t0) {
			t.Fatalf("trận %d phải được tính", i+1)
		}
	}
	if a.countPKWin("doi_thu", t0) || !a.countPKWin("nguoi_khac", t0) {
		t.Fatal("chỉ chặn cặp đã đủ, đối thủ khác vẫn tính")
	}
}

// Luật 4: thắng vì đối thủ bỏ trận không cộng số thắng (danh hiệu, bảng tuần, điểm hội).
func TestLeaverWinNotRanked(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, tokA := e.playerTok(t, "alice", "Gà Bông")
	aid, _ := e.store.AccountIDForToken(tokA)
	b, _ := e.playerTok(t, "bob", "Gà Mơ")
	e.warp(a, "arena")
	e.warp(b, "arena")
	var bPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Mơ" {
			bPID = p["pid"].(string)
			break
		}
	}
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "peck"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})
	a.expect("pk_go", 6*time.Second)
	r := e.roomOf(t)
	r.DoWait(func() {
		for _, q := range r.players {
			if q.PID == bPID {
				r.endMatchFor(q, "left")
			}
		}
	})
	if end := a.expect("pk_end", 3*time.Second); !strings.Contains(end["reason"].(string), "rời") {
		t.Fatalf("phải xử bỏ trận: %v", end)
	}
	acc, _ := e.store.Account(aid)
	var wins, played int
	r.DoWait(func() { wins, played = acc.Stats.PKWins, acc.Stats.PKPlayed })
	if wins != 0 || played != 1 {
		t.Fatalf("thắng vì bỏ trận: tính thắng %d (muốn 0), đã chơi %d", wins, played)
	}
}

// M10: hết đếm ngược mà một bên vẫn mất kết nối thì huỷ trận (nối lại kịp thì vẫn đấu, xem TestReconnectDuringPK).
func TestPKCountdownDisconnectCancels(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	b, _ := e.playerTok(t, "bob", "Gà Mơ")
	e.warp(a, "arena")
	e.warp(b, "arena")
	var bPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Mơ" {
			bPID = p["pid"].(string)
			break
		}
	}
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "peck"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})
	a.expect("pk_start", 2*time.Second)
	b.conn.Close()
	if end := a.expect("pk_end", 6*time.Second); end["cancelled"] != true {
		t.Fatalf("mất mạng lúc đếm ngược phải huỷ trận: %v", end)
	}
}

// M7: người được đề nghị làm chủ hội rời hội thì đề nghị mất hiệu lực.
func TestGuildTransferClearedOnLeave(t *testing.T) {
	gs, _ := OpenGuilds(t.TempDir())
	owner, vice := strings.Repeat("2", AccountIDLen), strings.Repeat("3", AccountIDLen)
	gd, _ := gs.Create(owner, "Chủ", "Hội Chuyển", "", "")
	gs.mu.Lock()
	_ = gs.addMemberLocked(gs.Guilds[gd.ID], vice, "Phó")
	gs.Guilds[gd.ID].Transfer, gs.Guilds[gd.ID].TransferAt = vice, time.Now()
	gs.mu.Unlock()
	if err := gs.Leave(vice); err != nil {
		t.Fatal(err)
	}
	if g := gs.View(owner); g.Transfer != "" {
		t.Fatalf("đề nghị chuyển quyền còn treo cho người đã rời: %q", g.Transfer)
	}
}

// M8: đăng nhập là có mặt: chủ hội chơi hằng ngày không bị coi là vắng.
func TestGuildOwnerTouchedOnLogin(t *testing.T) {
	e := newEnv(t, t.TempDir())
	_, id := e.seeded(t, "chuhoi", "Gà Chủ Hội", func(*Account) {})
	gd, err := e.store.Guilds.Create(id, "Gà Chủ Hội", "Hội Có Mặt", "", "")
	if err != nil {
		t.Fatal(err)
	}
	e.store.Guilds.mu.Lock()
	e.store.Guilds.Guilds[gd.ID].Members[id].LastSeen = time.Now().Add(-30 * 24 * time.Hour)
	e.store.Guilds.mu.Unlock()
	tok, _ := e.auth(t, "/api/login", "chuhoi", "matkhau1")
	c := e.dial(t, "chuhoi2", tok)
	c.expect("welcome", 2*time.Second)
	if g := e.store.Guilds.View(id); time.Since(g.Members[id].LastSeen) > time.Minute {
		t.Fatalf("đăng nhập phải cập nhật lần có mặt: %v", g.Members[id].LastSeen)
	}
}

// M9: khu đã đủ người thì không bước vào được (khu ngoài trời không giới hạn riêng).
func TestAreaCapacityGate(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, aid := e.seeded(t, "trongvuon", "Gà Trong Vườn", func(*Account) {})
	_, bid := e.seeded(t, "ngoaivuon", "Gà Ngoài Vườn", func(*Account) {})
	e.warp(a, "garden")
	old := Zones["garden"].Capacity
	Zones["garden"].Capacity = 1
	t.Cleanup(func() { Zones["garden"].Capacity = old })
	r := e.roomOf(t)
	var full, outdoor bool
	r.DoWait(func() {
		var pa, pb *Player
		for _, q := range r.players {
			switch q.acc.ID {
			case aid:
				pa = q
			case bid:
				pb = q
			}
		}
		full = r.areaFull(pb, "garden")
		outdoor = r.areaFull(pa, r.world.Outdoor)
	})
	if !full || outdoor {
		t.Fatalf("vườn đầy phải chặn (%v), ngoài trời không chặn (%v)", full, outdoor)
	}
}

// M11: danh hiệu tuần chưa nhận được giữ lại dù người thắng vắng cả tuần kế tiếp.
func TestWeeklyTitleOwedAcrossWeeks(t *testing.T) {
	dir := t.TempDir()
	w := OpenWeekly(dir)
	mon := time.Date(2026, 11, 2, 9, 0, 0, 0, vnZone)
	w.Submit("longest", "vang_mat", "Gà Vắng", 30, "Giun vàng", mon)
	w.Submit("longest", "khac", "Gà Khác", 20, "Giun đất", mon.AddDate(0, 0, 7)) // tuần sau có người khác đứng đầu
	later := mon.AddDate(0, 0, 14)
	if got := OpenWeekly(dir).TitlesFor("vang_mat", later); len(got) != 1 {
		t.Fatalf("hai tuần sau vẫn phải nhận được danh hiệu: %+v", got)
	}
	w2 := OpenWeekly(dir)
	w2.Claimed("vang_mat", "longest")
	if got := w2.TitlesFor("vang_mat", later); len(got) != 0 {
		t.Fatal("đã nhận thì không trao lại")
	}
}

// L2: kích thước giun theo bậc hiếm (huyền thoại tối đa ×1,6).
func TestWormSizeByTier(t *testing.T) {
	var legend *Worm
	for i := range Worms {
		if wormTier(&Worms[i]) == 4 {
			legend = &Worms[i]
			break
		}
	}
	if legend == nil {
		t.Skip("không có giun huyền thoại trong bảng")
	}
	a := newAcc()
	loot, _, _ := a.FinishDigWorm(legend, "garden", "dat_a", false, 1, t0, testBalance(), 1)
	if loot.SizeCM != 48 { // (8 + 22) × 1,6
		t.Fatalf("giun huyền thoại dài nhất %.1f cm, muốn 48", loot.SizeCM)
	}
}

// Lỗi thấp: lượt đào kéo qua nửa đêm được tính cho ngày mới, không bị xoá khi đổi ngày.
func TestDigAcrossMidnightCounted(t *testing.T) {
	a := newAcc()
	a.EnsureDaily(t0)
	a.DailyDigs = 50
	next := t0.Add(24 * time.Hour)
	a.FinishDigWorm(&Worms[0], "garden", "dat_a", false, 0.5, next, testBalance(), 1)
	a.EnsureDaily(next)
	if a.DailyDigs != 1 {
		t.Fatalf("ngày mới đếm %d lượt, muốn 1", a.DailyDigs)
	}
}

// L9: DoWait trên phòng đã dừng trả về ngay, không treo.
func TestDoWaitOnStoppedRoom(t *testing.T) {
	e := newEnv(t, t.TempDir())
	r := NewRoom(e.hub, "thu", "Phòng thử")
	go r.Run()
	r.DoWait(func() { r.quit = true })
	done := make(chan struct{})
	go func() { r.DoWait(func() {}); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("DoWait treo trên phòng đã dừng")
	}
}

// L11: cứu đói không mở khi còn nông sản/cá ăn hoặc bán được, và tối đa RescuePerDay lần/ngày.
func TestRescueRules(t *testing.T) {
	a := newAcc()
	a.Mana, a.Thoc = 5, 0
	a.Items["ca_ro"] = 1
	if a.RescueEligible() {
		t.Fatal("còn cá thì không được cứu đói")
	}
	a.Items = map[string]int{}
	b := testBalance()
	for i := 0; i < RescuePerDay; i++ {
		job, err := a.StartRescue(t0)
		if err != nil {
			t.Fatalf("lần %d: %v", i+1, err)
		}
		if err := a.FinishRescue(job.ID, t0.Add(b.RescueDuration), b); err != nil {
			t.Fatalf("xong lần %d: %v", i+1, err)
		}
		a.Thoc = 0 // vẫn đói, vẫn không có gì
	}
	if _, err := a.StartRescue(t0); err == nil {
		t.Fatal("quá số lần trong ngày phải chặn")
	}
}

// L12: hoàn nguyên giao dịch giữ dòng sổ cái của thay đổi trước đó (chưa ghi đĩa), chỉ bỏ dòng của giao dịch hỏng.
func TestRestoreKeepsEarlierLedger(t *testing.T) {
	a := newAcc()
	_ = a.addThoc(10, "truoc", "x", t0)
	snap, _ := snapshotOf(a)
	_ = a.addThoc(5, "hong", "y", t0)
	restore(a, snap)
	if l := a.TakeLedger(); len(l) != 1 || a.Thoc != 10 {
		t.Fatalf("sổ cái sau hoàn nguyên: %d dòng, ví %d", len(l), a.Thoc)
	}
}

// Lỗi thấp: giao dịch không vượt giới hạn túi 60 giun của người nhận.
func TestTradeRespectsBagCap(t *testing.T) {
	w := &Worms[0]
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "tga", "Gà Đưa Giun", func(acc *Account) { acc.Level = 3; acc.Worms[w.ID] = 5 })
	b, bid := e.seeded(t, "tgb", "Gà Túi Đầy", func(acc *Account) { acc.Level = 3; acc.Worms[w.ID] = BagCapacity - 2 })
	a.expect("pjoin", 2*time.Second)
	pos := a.position()
	b.walkTo(pos.X+1, pos.Z)
	id := openTradeWith(t, e, a, b, "Gà Túi Đầy")
	a.mustOK("trade_offer", msg{"id": id, "offer": map[string]int{w.ID: 5}})
	a.mustOK("trade_lock", msg{"id": id})
	b.mustOK("trade_lock", msg{"id": id})
	a.mustOK("trade_confirm", msg{"id": id})
	if r := b.act("trade_confirm", msg{"id": id}); r["ok"] == true || !strings.Contains(r["err"].(string), "túi") {
		t.Fatalf("túi người nhận sẽ vượt %d: %v", BagCapacity, r)
	}
	acc, _ := e.store.Account(bid)
	var n int
	e.roomOf(t).DoWait(func() { n = acc.Worms[w.ID] })
	if n != BagCapacity-2 {
		t.Fatalf("huỷ thì túi giữ nguyên: %d", n)
	}
}
