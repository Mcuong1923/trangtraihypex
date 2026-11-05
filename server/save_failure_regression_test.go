package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Test hồi quy cho nhóm "mất tiền, mất đồ" (M1, M2, M4–M6, M12–M14) trong
// plans/reports/code-reviewer-261105-1412-game-logic-review.md.

// blockWrite làm hỏng lần ghi kế tiếp của writeAtomic vào path: đặt một thư mục trùng tên file tạm ".tmp".
func blockWrite(t *testing.T, path string) func() {
	t.Helper()
	if err := os.MkdirAll(path+".tmp", 0o755); err != nil {
		t.Fatal(err)
	}
	return func() { os.RemoveAll(path + ".tmp") }
}

// playerOf trả Player của tài khoản id trong phòng công khai đầu tiên.
func (e *env) playerOf(t *testing.T, id string) (*Room, *Player) {
	t.Helper()
	r := e.roomOf(t)
	var p *Player
	r.DoWait(func() {
		for _, x := range r.players {
			if x.acc.ID == id {
				p = x
			}
		}
	})
	if p == nil {
		t.Fatal("không thấy người chơi trong phòng")
	}
	return r, p
}

// M1: lên cấp nhờ ăn giun (nguồn XP chính) thì nhận quà cấp ngay.
func TestEatWormLevelUpGivesReward(t *testing.T) {
	w := &Worms[0]
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "anlencap", "Gà Ăn Lên Cấp", func(acc *Account) {
		acc.Worms[w.ID], acc.Thoc, acc.Mana, acc.XP = 1, 100, 10, LevelXP(1)-1
	})
	r := a.mustOK("eat_worm", msg{"id": w.ID})
	if r["data"].(map[string]any)["levelUp"].(float64) < 1 {
		t.Fatal("phải lên cấp")
	}
	if rw := a.expect("reward", 2*time.Second); rw["level"].(float64) != 2 {
		t.Fatalf("quà cấp 2 phải tới ngay: %v", rw)
	}
}

// M14: chốt lượt đào mà lưu tài khoản lỗi thì phiên vẫn bị đóng và tính hồi đất (không đào lại ngay để gieo lại loài).
func TestDigFinishSaveFailureKeepsCooldown(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.DigCooldown, b.DigCooldownSec = 5*time.Second, 5 })
	a, id := e.seeded(t, "daoloi", "Gà Đào Lỗi", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "garden")
	a.walkToItem("garden", "dat_a")
	start := a.mustOK("dig_start", msg{"id": "dat_a"})
	plan := a.tugPlan(start, true)
	unblock := blockWrite(t, e.store.accountPath(id))
	if r := a.act("dig_finish", plan); r["ok"] == true {
		t.Fatal("lưu lỗi thì không được báo thành công")
	}
	unblock()
	r, p := e.playerOf(t, id)
	var open bool
	var ready time.Time
	r.DoWait(func() { open, ready = p.dig != nil, p.acc.DigReadyAt })
	if open || !ready.After(time.Now()) {
		t.Fatalf("phiên đào còn mở %v, hồi đất tới %v", open, ready)
	}
	if res := a.act("dig_start", msg{"id": "dat_a"}); res["ok"] == true {
		t.Fatal("đang hồi đất thì chưa đào lại được")
	}
}

// M4: lập hội mà trừ phí không lưu được thì hội không được tồn tại (không có hội miễn phí).
func TestGuildCreateRolledBackOnSaveFailure(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, id := e.seeded(t, "lapcho", "Gà Lập Hội", func(acc *Account) { acc.Level, acc.Thoc = GuildLevel, GuildCost+100 })
	unblock := blockWrite(t, e.store.accountPath(id))
	if r := a.act("guild_create", msg{"text": "Hội Thử Lỗi"}); r["ok"] == true {
		t.Fatal("lưu lỗi thì không được báo thành công")
	}
	unblock()
	if gid, _, _ := e.store.Guilds.GuildOf(id); gid != "" {
		t.Fatal("hội phải bị giải thể khi phí chưa trừ")
	}
	a.mustOK("guild_create", msg{"text": "Hội Thử Lỗi"}) // tên vẫn dùng lại được
}

// M4: mua đất mà trừ thóc không lưu được thì nhà trở về như cũ.
func TestLandBuyRolledBackOnSaveFailure(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, id := e.seeded(t, "muadat", "Gà Mua Đất", func(acc *Account) { acc.Level, acc.Thoc = 3, 5000 })
	e.warp(a, "neighborhood")
	a.walkToItem("neighborhood", "chu_dau_tu")
	unblock := blockWrite(t, e.store.accountPath(id))
	if r := a.act("land_buy", msg{"i": 1, "mode": "duong"}); r["ok"] == true {
		t.Fatal("lưu lỗi thì không được báo thành công")
	}
	unblock()
	h, _ := e.store.Homes.Get(id, "Gà Mua Đất")
	if h.Tier != 0 {
		t.Fatalf("đất phải về hạng cũ, đang %d", h.Tier)
	}
	if a.num("thoc") != 5000 {
		t.Fatal("không trừ thóc")
	}
}

// M5: hợp đồng xong mà lưu lỗi thì không để lại khoản công (tránh trả hai lần).
func TestContractWorkSaveFailureDropsCredit(t *testing.T) {
	dir := t.TempDir()
	cs, err := OpenContracts(dir)
	if err != nil {
		t.Fatal(err)
	}
	owner, worker := strings.Repeat("a", AccountIDLen), strings.Repeat("b", AccountIDLen)
	k, err := cs.Create(owner, "Chủ", "do", 100, 1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.Take(k.ID, worker, "Thợ"); err != nil {
		t.Fatal(err)
	}
	unblock := blockWrite(t, cs.path)
	if _, err := cs.Work(k.ID, worker, owner, time.Now()); err == nil {
		t.Fatal("lưu lỗi phải báo lỗi")
	}
	unblock()
	if len(cs.Credits[worker]) != 0 || cs.Contracts[k.ID].State != "taken" {
		t.Fatalf("còn khoản công %v, trạng thái %s", cs.Credits[worker], cs.Contracts[k.ID].State)
	}
	if done, err := cs.Work(k.ID, worker, owner, time.Now().Add(time.Minute)); err != nil || done.State != "done" || len(cs.Credits[worker]) != 1 {
		t.Fatalf("làm lại sau khi lưu được: %v %v", done, err)
	}
	// M4: hợp đồng của giao dịch lỗi bị bỏ, không hoàn ký quỹ.
	k2, _ := cs.Create(strings.Repeat("c", AccountIDLen), "Chủ 2", "do", 100, 1, time.Hour)
	cs.Discard(k2.ID)
	if cs.Contracts[k2.ID] != nil || len(cs.Credits[strings.Repeat("c", AccountIDLen)]) != 0 {
		t.Fatal("bỏ hợp đồng không được hoàn ký quỹ")
	}
}

// M6: góp quỹ hội mà lưu hội lỗi thì quỹ trong bộ nhớ không được tăng.
func TestGuildFundRolledBackOnSaveFailure(t *testing.T) {
	gs, err := OpenGuilds(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	owner := strings.Repeat("d", AccountIDLen)
	gd, err := gs.Create(owner, "Chủ Hội", "Hội Quỹ", "", "")
	if err != nil {
		t.Fatal(err)
	}
	unblock := blockWrite(t, gs.path)
	if _, err := gs.AddFund(owner, 50); err == nil {
		t.Fatal("lưu lỗi phải báo lỗi")
	}
	unblock()
	if g := gs.View(owner); g == nil || g.Fund != 0 || g.Members[owner].Contributed != 0 {
		t.Fatalf("quỹ phải giữ nguyên: %+v", g)
	}
	if gid, _, _ := gs.GuildOf(owner); gid != gd.ID {
		t.Fatal("bảng thành viên phải dựng lại đúng")
	}
}

// M12: điểm danh khi kho đầy vẫn giữ chuỗi ngày, vật phẩm vào hàng chờ.
func TestCheckInWithFullStorage(t *testing.T) {
	a := newAcc()
	a.Items["rom"] = StorageCap
	a.CheckIn = CheckIn{Day: DayKey(t0.AddDate(0, 0, -1)), Streak: 1} // hôm nay là ngày 2: quà có hạt cải
	day, rw, err := a.DoCheckIn(t0)
	if err != nil || day != 2 || !rw.Queued || a.PendingItems["hat_cai"] != 2 || a.CheckIn.Streak != 2 {
		t.Fatalf("điểm danh ngày %d: %v, chờ %v, chuỗi %d", day, err, a.PendingItems, a.CheckIn.Streak)
	}
}

// M2: đổi lô làm luống rau về kho thì cây không mồ côi: chưa chín trả lại hạt (và phân ủ đã bón).
func TestLotChangeUprootsHomeBed(t *testing.T) {
	h := NewHome(strings.Repeat("e", AccountIDLen), "Gà Trồng")
	a := newAcc()
	a.Thoc, a.Level = 5000, 3
	cx, cz := coopCell(LandTiers[1]) // ô sẽ thành nền chuồng ở lô mới
	h.Objects = append(h.Objects, PlacedObj{ID: "l1", Kind: "luong_rau", X: cx, Z: cz})
	a.Beds["home:l1"] = &Plant{Seed: "hat_cai", At: t0, Ready: t0.Add(time.Hour), Wilt: t0.Add(3 * time.Hour), Fertile: true}
	if _, _, err := h.BuyLot(a, 1, "duong", t0); err != nil {
		t.Fatal(err)
	}
	if a.Beds["home:l1"] != nil || a.Items["hat_cai"] != 1 || a.Items["phan_u"] != 1 || h.Storage["luong_rau"] != 1 {
		t.Fatalf("luống về kho: cây %v, hạt %d, phân %d", a.Beds["home:l1"], a.Items["hat_cai"], a.Items["phan_u"])
	}
}

// Review: closure của withHome panic thì caller vẫn nhận lỗi, không treo phòng gọi (M13).
func TestWithHomePanicDoesNotHang(t *testing.T) {
	e := newEnv(t, t.TempDir())
	_, id := e.seeded(t, "nhapanic", "Gà Nhà Panic", func(*Account) {})
	home, _ := e.store.Homes.Get(id, "Gà Nhà Panic")
	room := NewHomeRoom(e.hub, home)
	e.hub.mu.Lock()
	e.hub.homes[id] = room
	e.hub.mu.Unlock()
	go room.Run()
	done := make(chan error, 1)
	go func() { done <- e.hub.withHome(id, "Gà Nhà Panic", func(*Home) error { panic("hỏng") }) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("panic phải trả lỗi")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("withHome treo khi closure panic")
	}
}

// Review: hoàn tác góp quỹ khi ghi hội lỗi vẫn giữ phần trừ trong bộ nhớ (không cho quỹ tiền miễn phí).
func TestUndoFundKeepsDeductionOnSaveFailure(t *testing.T) {
	gs, _ := OpenGuilds(t.TempDir())
	owner := strings.Repeat("f", AccountIDLen)
	if _, err := gs.Create(owner, "Chủ", "Hội Hoàn", "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := gs.AddFund(owner, 30); err != nil {
		t.Fatal(err)
	}
	unblock := blockWrite(t, gs.path)
	gs.UndoFund(owner, 30)
	unblock()
	if g := gs.View(owner); g.Fund != 0 || g.Members[owner].Contributed != 0 {
		t.Fatalf("quỹ sau hoàn tác: %d, góp %d", g.Fund, g.Members[owner].Contributed)
	}
}

// Review: gửi RID của kết quả cũ lúc đang có phiên đào mới chỉ nhận lại kết quả cũ; phiên mới không bị đóng mà
// né hồi đất (act trả kết quả cũ trước mọi xử lý, phiên còn mở thì tự hết hạn và tính hồi).
func TestDigFinishReplayedRIDKeepsCooldown(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.DigCooldown, b.DigCooldownSec = 2*time.Second, 2 })
	a, id := e.seeded(t, "daocu", "Gà RID Cũ", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "garden")
	a.walkToItem("garden", "dat_a")
	first := a.mustOK("dig_start", msg{"id": "dat_a"})
	if r := a.actRID("rid-cu", "dig_finish", a.tugPlan(first, true)); r["ok"] != true {
		t.Fatal(r["err"])
	}
	time.Sleep(2100 * time.Millisecond)
	second := a.mustOK("dig_start", msg{"id": "dat_a"})
	a.actRID("rid-cu", "dig_finish", msg{"id": second["data"].(map[string]any)["id"]}) // phiên mới vẫn mở, gửi kèm RID cũ
	r, p := e.playerOf(t, id)
	var open bool
	var ready time.Time
	r.DoWait(func() { open, ready = p.dig != nil, p.acc.DigReadyAt })
	if !open && !ready.After(time.Now()) {
		t.Fatalf("RID cũ đã đóng phiên mà không tính hồi đất (tới %v)", ready)
	}
}

// Review: xây nhà dọn nền có luống đang trồng thì trả lại hạt, không để cây mồ côi (M2).
func TestBuildHouseUprootsBedOnFoot(t *testing.T) {
	h := NewHome(strings.Repeat("9", AccountIDLen), "Gà Xây")
	h.Tier = 1
	a := newAcc()
	a.Thoc = 5000
	a.Items["go"], a.Items["dinh"] = 20, 10
	x0, _, z0, _ := h.houseFoot()
	h.Objects = append(h.Objects, PlacedObj{ID: "l9", Kind: "luong_ca_rot", X: x0, Z: z0})
	a.Beds["home:l9"] = &Plant{Seed: "hat_ca_rot", At: t0, Ready: t0.Add(time.Hour), Wilt: t0.Add(3 * time.Hour)}
	if _, err := h.BuildHouse(a, "chuong_nho", "go", "do", t0, 1); err != nil {
		t.Fatal(err)
	}
	if a.Beds["home:l9"] != nil || a.Items["hat_ca_rot"] != 1 {
		t.Fatalf("xây nhà: cây %v, hạt %d", a.Beds["home:l9"], a.Items["hat_ca_rot"])
	}
}
