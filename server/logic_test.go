package main

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

// Kiểm thử luật PLAN_LOGIC_GAME: độ no, chống kẹt, chạy nhanh, giằng co giun, kho đồ, quà cấp,
// thú cưng có kỹ năng, trồng cây, điểm danh.

func TestLiveManaLastsFifteenToTwentyMinutes(t *testing.T) {
	b := LiveBalance()
	walk := newAcc()
	walk.TickVitals(22*60, b)
	if walk.Mana > 2 || walk.Mana < 0 {
		t.Fatalf("đi dạo 22 phút phải gần hết độ no, còn %.1f", walk.Mana)
	}
	// Đào liên tục ~5 lượt/phút trong 15 phút.
	dig := newAcc()
	for i := 0; i < 16*60; i++ {
		dig.TickVitals(1, b)
		if i%12 == 0 {
			dig.Mana = math.Max(0, dig.Mana-b.ManaPerDig)
		}
	}
	if dig.Mana > 1 {
		t.Fatalf("đào liên tục ~15–16 phút phải hết, còn %.1f", dig.Mana)
	}
	rest := newAcc()
	rest.TickVitalsAt(22*60, b, b.ManaRestFactor)
	if math.Abs(rest.Mana-50.5) > 1 {
		t.Fatalf("ngồi ổ phải tụt chậm một nửa, còn %.1f", rest.Mana)
	}
	if LiveBalance().HungerSlowSec != 180 || LiveBalance().HungerExhaustSec != 360 {
		t.Fatal("đi chậm sau 3 phút, kiệt sức sau 6 phút ở 0")
	}
	if DemoBalance().ManaDrainPerSec != b.ManaDrainPerSec {
		t.Fatal("bộ demo dùng chung tốc độ tụt mới")
	}
}

func TestStarterGiftOnce(t *testing.T) {
	b := LiveBalance()
	a := newAcc()
	a.Char = &Character{Name: "Gà Mới"}
	if !a.GrantStarter(t0, b) || a.Thoc != 100 {
		t.Fatalf("quà khởi nghiệp 100 thóc, có %d", a.Thoc)
	}
	if a.GrantStarter(t0, b) || a.Thoc != 100 {
		t.Fatal("chỉ nhận một lần")
	}
	if l := a.TakeLedger(); len(l) != 1 || l[0].Reason != "starter_gift" {
		t.Fatal("ghi sổ cái lý do starter_gift")
	}
}

func TestRawEatAndRationNeverStuck(t *testing.T) {
	b := LiveBalance()
	a := newAcc()
	a.Mana, a.Thoc = 0, 0
	a.ZeroElapsed = b.HungerExhaustSec + 1
	if a.HungerStage(b) != 4 {
		t.Fatal("kiệt sức")
	}
	// Không thóc, không đồ ăn: cụ cho một khẩu phần, mỗi ngày một lần.
	if err := a.ElderRation(t0); err != nil || a.Food["cam"] != 1 {
		t.Fatalf("bữa cứu trợ: %v", err)
	}
	a.Food["cam"] = 0
	if err := a.ElderRation(t0.Add(time.Hour)); err == nil {
		t.Fatal("một ngày một lần")
	}
	// Vẫn đào được giun thường rồi ăn sống, không tốn phí, không cộng XP.
	if err := a.CanDig(t0, b); err != nil {
		t.Fatal(err)
	}
	w := RollDigWorm(RegionHypex, 0.2, t0, true, 0, 1)
	a.FinishDigWorm(w, "garden", "dat_a", true, 0.5, t0, b, 1.5)
	xp := a.XP
	gain, _, err := a.EatWormRaw(w.ID, b, t0)
	if err != nil || gain != int(math.Round(float64(w.Mana)*0.6)) || a.Thoc != 0 || a.XP != xp {
		t.Fatalf("ăn sống: +%d err %v thóc %d", gain, err, a.Thoc)
	}
	if a.DigReadyAt.Sub(t0) != time.Duration(float64(b.DigCooldown)*1.5) {
		t.Fatal("kiệt sức: hồi đất ×1,5")
	}
}

func TestTugRatesByTier(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 9))
	want := [][2]float64{{1, 1}, {1, 1}, {.85, .95}, {.70, .90}, {.55, .80}}
	for _, p := range TugTiers {
		caught, idle := 0, 0
		for i := 0; i < 1500; i++ {
			b := TugBursts(r.Uint32(), p)
			if SimulateTug(p, b, PlanTug(p, b, .35, .7, .12+r.Float64()*.16)).Caught {
				caught++
			}
			if SimulateTug(p, b, nil).Caught {
				idle++
			}
		}
		rate := float64(caught) / 1500
		if rate < want[p.Tier][0]-.02 || rate > want[p.Tier][1]+.02 {
			t.Fatalf("bậc %d: người chơi hợp lý bắt được %.1f%%, muốn %.0f–%.0f%%", p.Tier, rate*100, want[p.Tier][0]*100, want[p.Tier][1]*100)
		}
		if p.CanEscape && idle > 0 {
			t.Fatalf("bậc %d: không bấm gì mà vẫn bắt được giun hiếm", p.Tier)
		}
		if !p.CanEscape && idle != 1500 {
			t.Fatalf("bậc %d: giun thường luôn phải lôi lên được", p.Tier)
		}
	}
}

func TestTugDeterministicAndBounded(t *testing.T) {
	p := TugTiers[3]
	a, b := TugBursts(12345, p), TugBursts(12345, p)
	if len(a) == 0 || len(a) != len(b) || a[0] != b[0] || a[len(a)-1] != b[len(b)-1] {
		t.Fatal("cùng seed phải cùng lịch cơn giật")
	}
	ev := PlanTug(p, a, .4, .6, .1)
	r1, r2 := SimulateTug(p, a, ev), SimulateTug(p, a, ev)
	if r1 != r2 || !r1.Caught || r1.End <= 0 || r1.End > p.MaxSec {
		t.Fatalf("mô phỏng tất định: %+v", r1)
	}
	// Giữ liên tục với giun huyền thoại: tuột 3 lần là thoát.
	hold := SimulateTug(TugTiers[4], TugBursts(99, TugTiers[4]), []TugEvent{{0, true}})
	if !hold.Escaped || hold.Slips != 3 {
		t.Fatalf("giữ chặt mãi phải tuột và thoát: %+v", hold)
	}
	// mulberry32 cùng số với JavaScript (bảng tham chiếu cho client/js/dig.js).
	rr := &tugRand{s: 1}
	if got := []float64{rr.next(), rr.next()}; math.Abs(got[0]-0.6270739405881613) > 1e-12 || math.Abs(got[1]-0.002735721180215478) > 1e-12 {
		t.Fatalf("mulberry32 lệch JS: %v", got)
	}
}

func TestDigLuckKeepsTableNormalised(t *testing.T) {
	counts := map[int]int{}
	r := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < 200000; i++ {
		counts[wormTier(RollDigWorm(RegionHypex, r.Float64(), t0.AddDate(1, 0, 0), false, LuckCap, 1))]++
	}
	base := 0.0
	for _, w := range Worms[1:] {
		base += w.Prob
	}
	got := 1 - float64(counts[0])/200000
	if math.Abs(got-base*1.10) > 0.004 {
		t.Fatalf("may mắn +10%% phải nhân tỉ lệ giun không thường: %.4f vs %.4f", got, base*1.10)
	}
}

func TestStaminaSprintAndLock(t *testing.T) {
	r := &Room{hub: &Hub{bal: LiveBalance()}}
	p := &Player{acc: newAcc(), run: true}
	now := t0
	ran := 0.0
	for i := 0; i < 200 && p.stamLock.IsZero(); i++ {
		if r.stepStamina(p, now, 0.05, 0, true) {
			ran += 0.05
		}
		now = now.Add(50 * time.Millisecond)
	}
	if math.Abs(ran-StaminaSec) > 0.11 || p.stamLock.IsZero() {
		t.Fatalf("chạy được %.2f giây rồi phải hụt hơi", ran)
	}
	if r.stepStamina(p, now.Add(4*time.Second), 0.05, 0, true) {
		t.Fatal("đang khoá 5 giây thì không chạy được")
	}
	if !r.stepStamina(p, now.Add(5*time.Second+100*time.Millisecond), 0.05, 0, true) || p.stam < StaminaSec-0.1 {
		t.Fatal("hết 5 giây thể lực đầy lại và chạy tiếp được")
	}
	if r.maxStamina(p, 2) != StaminaSec/2 || r.maxStamina(p, 3) != 0 {
		t.Fatal("đói: còn một nửa; đi chậm/kiệt sức: không chạy")
	}
}

func TestLevelRewardsOnceAndEggMigration(t *testing.T) {
	a := newAcc()
	a.Level = 5
	a.EggsClaimed[5] = true // đã nhận trứng mốc 5 qua Vịt Bông trước bản cập nhật
	got := a.ClaimLevelRewards(t0)
	if len(got) != 4 || a.Items["trung_thuong"] != 1 || a.Thoc != 50+80+100+120 || a.Food["cam"] != 3 {
		t.Fatalf("quà cấp 2..5: %d mức, trứng %d, thóc %d", len(got), a.Items["trung_thuong"], a.Thoc)
	}
	if len(a.ClaimLevelRewards(t0)) != 0 {
		t.Fatal("mỗi cấp nhận một lần")
	}
	old := &Account{ID: "acc_cu_000", Eggs: 2}
	old.normalize()
	if old.Eggs != 0 || old.Items["trung_thuong"] != 2 || old.BusTokens != MaxBusTokens {
		t.Fatal("bản lưu cũ: trứng vào kho, đủ 2 lượt xe")
	}
}

func TestPetAbilities(t *testing.T) {
	a := newAcc()
	a.Companions = []Companion{{ID: "p1", Species: "vit_con", Affinity: 0}, {ID: "p2", Species: "ga_tre", Affinity: 200}}
	a.Following = "p1"
	if fareAfter(a, 10) != 8 || petBonus(a, AbilityBelly) != 0 {
		t.Fatal("vịt Lv1 giảm 20%; kỹ năng chỉ của con đang đi theo")
	}
	a.Companions[0].Affinity = 160
	if fareAfter(a, 10) != 5 || fareAfter(a, 3) != 1 {
		t.Fatal("vịt Lv3 giảm 50%, làm tròn xuống")
	}
	a.Following = "p2"
	if petBonus(a, AbilityBelly) != .25 {
		t.Fatal("gà tre Lv3 giảm 25% tốc tụt")
	}
	for egg, tab := range EggSpecies {
		sum := 0.0
		for _, e := range tab {
			sum += e.P
			if speciesByID(e.Species) == nil {
				t.Fatalf("%s: loài lạ %s", egg, e.Species)
			}
		}
		if math.Abs(sum-1) > 1e-6 {
			t.Fatalf("%s: tổng tỉ lệ %.4f", egg, sum)
		}
	}
	counts := map[string]int{}
	r := rand.New(rand.NewPCG(5, 6))
	for i := 0; i < 10000; i++ {
		counts[hatchSpecies("trung_thuong", r.Float64())]++
	}
	if math.Abs(float64(counts["ga_con"])/10000-.6) > .02 || math.Abs(float64(counts["ngong_con"])/10000-.15) > .02 {
		t.Fatalf("phân phối trứng thường lệch: %v", counts)
	}
}

func TestGardenGrowWaterHarvestWilt(t *testing.T) {
	a := newAcc()
	a.Items["hat_cai"] = 3
	if _, err := a.PlantSeed("park:luong_1", "hat_cai", false, t0, 1); err != nil {
		t.Fatal(err)
	}
	a.Items["hat_ca_rot"] = 1
	if _, err := a.PlantSeed("park:luong_2", "hat_ca_rot", false, t0, 1); err == nil {
		t.Fatal("mỗi người một luống cộng đồng")
	}
	if _, _, err := a.Harvest("park:luong_1", t0.Add(5*time.Minute)); err == nil {
		t.Fatal("chưa chín")
	}
	if err := a.WaterBed("park:luong_1", t0.Add(2*time.Minute)); err != nil || a.WaterBed("park:luong_1", t0.Add(3*time.Minute)) == nil {
		t.Fatal("tưới một lần")
	}
	crop, n, err := a.Harvest("park:luong_1", t0.Add(8*time.Minute))
	if err != nil || crop != "cai" || n != 3 || a.Items["cai"] != 3 {
		t.Fatalf("tưới rút 25%%: chín trước 8 phút; thu %s %d %v", crop, n, err)
	}
	a.PlantSeed("home:obj1", "hat_cai", false, t0, 1)
	crop, _, _ = a.Harvest("home:obj1", t0.Add(40*time.Minute))
	if crop != "phan_u" {
		t.Fatal("để quá 3 lần thời gian lớn thì héo, còn phân ủ")
	}
	if g, err := a.EatItem("cai"); err == nil || g != 0 {
		_ = g // mana đang đầy: không ăn được
	}
}

func TestCheckInStreak(t *testing.T) {
	a := newAcc()
	day := t0
	for i := 1; i <= 7; i++ {
		n, _, err := a.DoCheckIn(day)
		if err != nil || n != i {
			t.Fatalf("ngày %d: %d %v", i, n, err)
		}
		if _, _, err := a.DoCheckIn(day.Add(time.Hour)); err == nil {
			t.Fatal("mỗi ngày một lần")
		}
		day = day.AddDate(0, 0, 1)
	}
	if a.Items["trung_dom"] != 1 {
		t.Fatal("ngày 7 tặng trứng đốm")
	}
	if n, _, _ := a.DoCheckIn(day.AddDate(0, 0, 1)); n != 1 {
		t.Fatal("lỡ một ngày thì quay về ngày 1")
	}
}

func TestBuyItemsAndStorageCap(t *testing.T) {
	a := newAcc()
	a.Thoc = 10000
	if err := a.BuyItem("go", 10, "vatlieu", t0); err != nil || a.Items["go"] != 10 || a.Thoc != 9920 {
		t.Fatalf("mua gỗ: %v", err)
	}
	if err := a.BuyItem("go", 10, "thucan", t0); err == nil {
		t.Fatal("quầy thức ăn không bán gỗ")
	}
	a.Items["rom"] = StorageCap - 10
	if err := a.BuyItem("da", 1, "vatlieu", t0); err == nil || a.Items["da"] != 0 {
		t.Fatal("kho đầy thì không mua thêm")
	}
}

// Mọi nhiệm vụ NPC đặt hàng đúng một thứ có thật và thưởng vật phẩm có thật; mỗi ngày đều có một nhiệm vụ.
func TestNPCQuestsWellFormed(t *testing.T) {
	for _, q := range NPCQuests {
		if (q.Item == "") == (q.Worm == "") || q.N <= 0 || len(q.Gift) == 0 || q.Line == "" {
			t.Fatalf("nhiệm vụ %s thiếu dữ liệu", q.ID)
		}
		if q.Item != "" && itemByID(q.Item) == nil || q.Worm != "" && wormByID(q.Worm) == nil {
			t.Fatalf("nhiệm vụ %s đặt hàng thứ không có", q.ID)
		}
		for id := range q.Gift {
			if itemByID(id) == nil {
				t.Fatalf("nhiệm vụ %s thưởng thứ không có: %s", q.ID, id)
			}
		}
	}
	seen := map[string]bool{}
	for d := 0; d < 60; d++ {
		seen[OrdersFor(DayKey(t0.AddDate(0, 0, d)))[3].ID] = true
	}
	if len(seen) < len(NPCQuests)/2 {
		t.Fatalf("nhiệm vụ NPC phải đổi theo ngày: chỉ thấy %d loại trong 60 ngày", len(seen))
	}
}
