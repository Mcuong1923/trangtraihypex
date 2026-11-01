package main

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 31, 9, 0, 0, 0, vnZone)

func newAcc() *Account {
	a := NewAccount("abcdef123456", "tester", t0)
	a.Char = &Character{Name: "Gà Thử"}
	return a
}

func TestWormTableSumsToOne(t *testing.T) {
	sum := 0.0
	for _, w := range Worms {
		sum += w.Prob
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("tổng xác suất = %v, cần 1", sum)
	}
}

func TestRollWormBoundaries(t *testing.T) {
	cases := map[float64]string{0: "dat", 0.7499: "dat", 0.75: "beo", 0.9499: "beo", 0.95: "sang", 0.9899: "sang", 0.99: "vang", 0.9989: "vang", 0.999: "cauvong", 0.99999: "cauvong"}
	for r, want := range cases {
		if got := RollWorm(r).ID; got != want {
			t.Errorf("RollWorm(%v) = %s, cần %s", r, got, want)
		}
	}
}

// Mô phỏng phân bố bằng seed cố định: không phải test ngẫu nhiên dễ lỗi vì xui (mục 20).
func TestRollWormDistribution(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	n := 200000
	count := map[string]int{}
	for i := 0; i < n; i++ {
		count[RollWorm(rng.Float64()).ID]++
	}
	for _, w := range Worms {
		got := float64(count[w.ID]) / float64(n)
		if math.Abs(got-w.Prob) > 0.01+w.Prob*0.15 {
			t.Errorf("%s: %.4f, kỳ vọng %.4f", w.ID, got, w.Prob)
		}
	}
	// Kỳ vọng thu nhập 20 lượt nếu bán hết = 192 thóc (mục 8.2).
	ev := 0.0
	for _, w := range Worms {
		ev += w.Prob * float64(w.Sell)
	}
	if math.Abs(ev*20-192) > 0.5 {
		t.Errorf("kỳ vọng 20 lượt = %.1f, plan ghi 192", ev*20)
	}
}

func TestLevelFormula(t *testing.T) {
	total := 0
	for l := 1; l < LevelCap; l++ {
		total += LevelXP(l)
	}
	if total != 2160 {
		t.Fatalf("tổng XP cấp 1→10 = %d, plan ghi 2.160", total)
	}
	a := newAcc()
	if gained := a.addXP(500); gained != 3 || a.Level != 4 || a.XP != 500-80-120-160 {
		t.Fatalf("addXP(500): +%d cấp, cấp %d, xp %d", gained, a.Level, a.XP)
	}
	a.addXP(100000)
	if a.Level != LevelCap || a.XP != 0 {
		t.Fatalf("trần cấp: %d/%d", a.Level, a.XP)
	}
}

func TestHungerStages(t *testing.T) {
	b := PlanBalance()
	a := newAcc()
	if a.HungerStage(b) != 0 {
		t.Fatal("bắt đầu phải bình thường")
	}
	a.Mana = 10
	if a.HungerStage(b) != 1 {
		t.Fatal("mana 10 phải là sắp đói")
	}
	a.Mana = 0.5
	a.TickVitals(60, b) // 1 điểm/phút → về 0
	if a.Mana != 0 || a.HungerStage(b) != 2 {
		t.Fatalf("mana %v stage %d", a.Mana, a.HungerStage(b))
	}
	a.TickVitals(300, b)
	if a.HungerStage(b) != 3 {
		t.Fatalf("5 phút tại 0 phải đói kéo dài, stage %d (zero %.0f)", a.HungerStage(b), a.ZeroElapsed)
	}
	a.TickVitals(300, b)
	if a.HungerStage(b) != 4 {
		t.Fatal("10 phút tại 0 phải kiệt sức")
	}
	if err := a.CanDig(t0, b); err == nil {
		t.Fatal("kiệt sức không được đào")
	}
	// Ăn dưới 20: giữ phạt, bộ đếm không reset.
	a.restoreMana(10)
	if a.HungerStage(b) != 4 {
		t.Fatal("hồi dưới 20 phải giữ mức phạt")
	}
	// Mana > 0 thì bộ đếm không tăng.
	z := a.ZeroElapsed
	a.TickVitals(10, b)
	if a.ZeroElapsed != z {
		t.Fatal("bộ đếm chỉ tăng khi mana = 0")
	}
	a.restoreMana(15)
	if a.HungerStage(b) != 0 || a.ZeroElapsed != 0 {
		t.Fatal("đạt 20 phải gỡ phạt và reset bộ đếm")
	}
}

func TestDigCostsManaAndCooldown(t *testing.T) {
	b := PlanBalance()
	a := newAcc()
	a.Mana = 50
	loot, _ := a.FinishDig("garden", "dat_a", true, 0.1, 0.5, t0, b)
	if loot.Worm != "dat" || a.Worms["dat"] != 1 || !a.Collection["dat"] {
		t.Fatalf("loot %+v túi %v", loot, a.Worms)
	}
	if a.Mana != 49 {
		t.Fatalf("mỗi lượt đào có thưởng trừ 1 mana, còn %v", a.Mana)
	}
	if a.CanDig(t0.Add(10*time.Second), b) == nil {
		t.Fatal("phải chờ 45 giây giữa hai phần thưởng")
	}
	if a.CanDig(t0.Add(46*time.Second), b) != nil {
		t.Fatal("sau 45 giây phải đào được")
	}
	a.DailyDigs = b.DigDailyLimit
	if a.CanDig(t0.Add(time.Hour), b) == nil {
		t.Fatal("quá 120 lượt/ngày phải bị chặn")
	}
	// Qua ngày (giờ Việt Nam) reset lượt.
	if a.CanDig(time.Date(2026, 11, 1, 0, 1, 0, 0, vnZone), b) != nil {
		t.Fatal("ngày mới phải reset giới hạn đào")
	}
}

func TestEatWormAtomic(t *testing.T) {
	a := newAcc()
	a.Worms["vang"] = 1
	a.Mana = 50
	a.Thoc = 3 // phí chế biến giun vàng là 8
	if _, _, err := a.EatWorm("vang", t0); err == nil {
		t.Fatal("thiếu thóc phải từ chối")
	}
	if a.Worms["vang"] != 1 || a.Thoc != 3 || a.Mana != 50 || a.XP != 0 {
		t.Fatal("thất bại không được làm mất giun/thóc")
	}
	a.Thoc = 10
	a.Mana = 100
	if _, _, err := a.EatWorm("vang", t0); err == nil {
		t.Fatal("mana đầy phải từ chối")
	}
	a.Mana = 80
	lv, _, err := a.EatWorm("vang", t0)
	// 10 − 8 phí + 20 thưởng nhiệm vụ ngày "ăn 1 con"; 200 XP = 80 + 120 → lên 2 cấp.
	if err != nil || a.Worms["vang"] != 0 || a.Thoc != 22 || a.Mana != 100 || lv != 2 {
		t.Fatalf("err %v túi %d thóc %d mana %v cấp +%d", err, a.Worms["vang"], a.Thoc, a.Mana, lv)
	}
	if _, _, err := a.SellWorm("vang", t0); err == nil {
		t.Fatal("cùng một con không được vừa ăn vừa bán")
	}
}

func TestDailyQuestsPayOnce(t *testing.T) {
	a := newAcc()
	a.Worms["dat"] = 5
	for i := 0; i < 5; i++ {
		a.SellWorm("dat", t0)
	}
	// 5 × 5 thóc + 20 thưởng nhiệm vụ bán 3 giun (chỉ trả một lần).
	if a.Thoc != 45 {
		t.Fatalf("thóc %d, cần 45", a.Thoc)
	}
	if len(a.TakeLedger()) != 6 {
		t.Fatal("mỗi biến động ví phải có một dòng sổ cái")
	}
}

func TestFoodAndCosmetics(t *testing.T) {
	a := newAcc()
	a.Thoc = 30
	a.Mana = 90
	if err := a.BuyFood("rau", true, t0); err != nil || a.Mana != 100 || a.Thoc != 18 {
		t.Fatalf("ăn đĩa rau: %v mana %v thóc %d", err, a.Mana, a.Thoc)
	}
	if err := a.BuyFood("cam", true, t0); err == nil {
		t.Fatal("không cho ăn khi đầy")
	}
	if a.Thoc != 18 {
		t.Fatal("từ chối thì không trừ thóc")
	}
	if err := a.BuyFood("cam", false, t0); err != nil || a.Food["cam"] != 1 {
		t.Fatal("mua dự trữ")
	}

	a.Thoc = 2000
	if err := a.BuyCosmetic("vest", t0); err != nil {
		t.Fatal(err)
	}
	if err := a.BuyCosmetic("vest", t0); err == nil || a.Thoc != 1100 {
		t.Fatal("không được mua trùng / trừ tiền hai lần")
	}
	if err := a.Equip("body", "vay_hong"); err == nil {
		t.Fatal("chỉ mặc món đã sở hữu")
	}
	if err := a.Equip("eye", "vest"); err == nil {
		t.Fatal("sai slot")
	}
	a.BuyCosmetic("vay_hong", t0) // váy và vest cùng slot thân: thay nhau
	if a.Equipped["body"] != "vay_hong" {
		t.Fatal("váy phải thay vest ở slot thân")
	}
	a.SaveOutfit(0)
	a.Equip("body", "vest")
	a.LoadOutfit(0)
	if a.Equipped["body"] != "vay_hong" {
		t.Fatal("tải bộ phối")
	}
}

func TestRescueJob(t *testing.T) {
	b := PlanBalance()
	a := newAcc()
	a.Mana = 0
	a.Thoc = 0
	job, err := a.StartRescue(t0)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.FinishRescue(job.ID, t0.Add(10*time.Second), b); err == nil {
		t.Fatal("chưa đủ thời gian")
	}
	if err := a.FinishRescue(job.ID, t0.Add(61*time.Second), b); err != nil || a.Thoc != 5 {
		t.Fatalf("hoàn thành: %v thóc %d", err, a.Thoc)
	}
	if err := a.FinishRescue(job.ID, t0.Add(70*time.Second), b); err == nil || a.Thoc != 5 {
		t.Fatal("không nhận thưởng hai lần cho một job")
	}
	// 5 thóc đủ mua khẩu phần +20 mana.
	if err := a.BuyFood("cam", true, t0); err != nil || a.Mana != 20 {
		t.Fatalf("mua khẩu phần sau cứu đói: %v", err)
	}
	a.Thoc = 50
	if _, err := a.StartRescue(t0); err == nil {
		t.Fatal("còn thóc thì không mở việc cứu đói")
	}
}

func TestShowcaseRules(t *testing.T) {
	b := PlanBalance()
	a := newAcc()
	common, _ := a.FinishDig("garden", "dat_a", false, 0.1, 0.5, t0, b)
	rare, _ := a.FinishDig("garden", "dat_a", false, 0.995, 0.5, t0, b)
	rare2, _ := a.FinishDig("garden", "dat_a", false, 0.96, 0.5, t0, b)
	if _, err := a.Showcase(common.ID, t0, b); err == nil {
		t.Fatal("giun thường không khoe")
	}
	if _, err := a.Showcase("loot_gia", t0, b); err == nil {
		t.Fatal("thẻ giả phải bị từ chối")
	}
	if _, err := a.Showcase(rare.ID, t0, b); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Showcase(rare.ID, t0.Add(2*time.Minute), b); err == nil {
		t.Fatal("mỗi lần đào chỉ khoe một lần")
	}
	if _, err := a.Showcase(rare2.ID, t0.Add(30*time.Second), b); err == nil {
		t.Fatal("tối đa 1 thẻ/60 giây")
	}
	if _, err := a.Showcase(rare2.ID, t0.Add(61*time.Second), b); err != nil {
		t.Fatal(err)
	}
}

func TestIdempotencyMemory(t *testing.T) {
	a := newAcc()
	a.Remember("r1", ActResult{OK: true, Msg: "x"})
	if r, ok := a.Remembered("r1"); !ok || r.Msg != "x" {
		t.Fatal("phải nhớ kết quả theo request ID")
	}
	for i := 0; i < 100; i++ {
		a.Remember(string(rune('A'+i)), ActResult{})
	}
	if len(a.Recent) > 64 {
		t.Fatal("bộ nhớ request ID phải có giới hạn")
	}
}

func TestValidateName(t *testing.T) {
	if _, err := ValidateName("A"); err == nil {
		t.Fatal("tên quá ngắn")
	}
	if n, err := ValidateName("  Gà   Bông "); err != nil || n != "Gà Bông" {
		t.Fatalf("chuẩn hoá khoảng trắng: %q %v", n, err)
	}
	if _, err := ValidateName("Admin Gà"); err == nil {
		t.Fatal("tên mạo danh quản trị phải bị chặn")
	}
}

func TestMarkerPos(t *testing.T) {
	if markerPos(0, 0.8) != 0 || math.Abs(markerPos(1.25, 0.8)-1) > 1e-9 || math.Abs(markerPos(2.5, 0.8)) > 1e-9 {
		t.Fatal("sóng tam giác canh lực")
	}
}
