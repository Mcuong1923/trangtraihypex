package main

import (
	"testing"
	"time"
)

func TestHomePlacementRules(t *testing.T) {
	h := NewHome("owner123456789012345678", "Gà Chủ")
	if len(h.Objects) != 1 || h.Storage["chau_hoa"] != 1 {
		t.Fatal("tổ khởi đầu: 1 ổ rơm đã đặt + 1 chậu hoa trong kho")
	}
	if _, err := h.Place("chau_hoa", 0, 3, 0); err == nil {
		t.Fatal("không được đặt chắn lối vào")
	}
	if _, err := h.Place("chau_hoa", 9, 0, 0); err == nil {
		t.Fatal("không được đặt ngoài đất")
	}
	if _, err := h.Place("chau_hoa", 1, -1, 0); err == nil {
		t.Fatal("ô đã có ổ rơm")
	}
	o, err := h.Place("chau_hoa", -2, -2, 5)
	if err != nil || o.Rot != 1 || h.Storage["chau_hoa"] != 0 {
		t.Fatalf("đặt hợp lệ, xoay 90° theo bội: %v %+v", err, o)
	}
	if _, err := h.Place("chau_hoa", 2, 0, 0); err == nil {
		t.Fatal("kho hết thì không đặt được")
	}
	if err := h.Pickup(o.ID); err != nil || h.Storage["chau_hoa"] != 1 {
		t.Fatal("cất đồ về kho")
	}
}

func TestHomeUpgradeAndBuild(t *testing.T) {
	h := NewHome("owner123456789012345678", "Gà Chủ")
	a := newAcc()
	a.Thoc = 5000
	if err := h.HireBuild(a, "do", t0, time.Minute); err == nil {
		t.Fatal("tổ nhỏ chưa xây chuồng được")
	}
	if _, _, err := h.BuyLot(a, h.Tier+1, "duong", t0); err == nil || a.Thoc != 5000 {
		t.Fatal("cấp 1 chưa được mua đất, không trừ tiền")
	}
	a.Level = 3
	// Đặt một món vào ô sẽ thành nền chuồng ở hạng đất mới để kiểm tra trả về kho.
	cx, cz := coopCell(LandTiers[1])
	h.Objects = append(h.Objects, PlacedObj{ID: "x", Kind: "thung_go", X: cx, Z: cz})
	moved, _, err := h.BuyLot(a, h.Tier+1, "duong", t0)
	if err != nil || h.Tier != 1 || a.Thoc != 3800 || moved != 1 || h.Storage["thung_go"] != 1 {
		t.Fatalf("nâng đất: %v tier %d thóc %d chuyển %d", err, h.Tier, a.Thoc, moved)
	}
	if err := h.HireBuild(a, "tim", t0, time.Minute); err == nil {
		t.Fatal("màu mái không hợp lệ")
	}
	if err := h.HireBuild(a, "xanh", t0, time.Minute); err == nil || a.Thoc != 3800 {
		t.Fatal("xây nhà cần vật liệu (PLAN_LOGIC_GAME 13.3), thiếu thì không trừ gì")
	}
	a.Items["go"], a.Items["dinh"] = 20, 10
	if err := h.HireBuild(a, "xanh", t0, time.Minute); err != nil || a.Thoc != 3400 || a.Items["go"] != 0 {
		t.Fatalf("thuê xây chuồng gỗ: %v %d", err, a.Thoc)
	}
	if err := h.HireBuild(a, "xanh", t0, time.Minute); err == nil {
		t.Fatal("không thuê xây hai lần")
	}
	if h.Tick(t0.Add(30*time.Second)) || h.Coop.State != "building" {
		t.Fatal("chưa tới giờ hoàn thành")
	}
	// Server tính theo giờ thật: chủ offline lâu vẫn hoàn thành đúng một lần.
	if !h.Tick(t0.Add(2*time.Hour)) || h.Coop.State != "done" || h.Tick(t0.Add(3*time.Hour)) {
		t.Fatal("hoàn thành công trình một lần")
	}
	if _, err := h.Place("thung_go", cx, cz, 0); err == nil {
		t.Fatal("không đặt đồ lên nền chuồng")
	}
	a.Thoc, a.Level = 100, 6
	if _, _, err := h.BuyLot(a, h.Tier+1, "duong", t0); err == nil || h.Tier != 1 {
		t.Fatal("thiếu thóc không được nâng đất")
	}
}

func TestLotsHouseDesignsAndUpgrade(t *testing.T) {
	h := NewHome("owner123456789012345678", "Gà Chủ")
	a := newAcc()
	a.Thoc, a.Level = 10000, 3
	if _, _, err := h.BuyLot(a, 2, "duong", t0); err == nil {
		t.Fatal("lô rộng cần cấp 6")
	}
	_, cost, err := h.BuyLot(a, 1, "ven_ao", t0)
	if err != nil || cost != 1200+300 || h.Tier != 1 || h.LotPlace != "ven_ao" || a.Thoc != 8500 {
		t.Fatalf("lô tiêu chuẩn ven ao: %v %d", err, cost)
	}
	if _, _, err := h.BuyLot(a, 1, "ven_ao", t0); err == nil {
		t.Fatal("không mua lại đúng lô đang có")
	}
	if _, err := h.BuildHouse(a, "nha_hai_gian", "go", "do", t0, 1); err == nil {
		t.Fatal("nhà hai gian cần lô rộng")
	}
	a.Items["rom"], a.Items["go"] = 20, 5
	dur, err := h.BuildHouse(a, "chuong_nho", "rom", "do", t0, 1)
	if err != nil || dur != 10*time.Minute || a.Thoc != 8300 {
		t.Fatalf("chuồng rơm 10 phút: %v %v", err, dur)
	}
	if regen, cap := h.comfort(); regen != 0 || cap != 8 {
		t.Fatal("đang xây chưa hồi độ no")
	}
	h.Tick(t0.Add(11 * time.Minute))
	if regen, cap := h.comfort(); regen != 1 || cap != 8 {
		t.Fatal("nhà rơm: +1/phút, 8 khách")
	}
	// Nâng lên gạch: chỉ trả phần chênh lệch vật liệu và nửa tiền công.
	a.Items = map[string]int{"gach": 20, "go": 10, "ngoi": 6}
	if _, err := h.UpgradeHouse(a, "gach", t0.Add(12*time.Minute), 1); err != nil || a.Thoc != 7900 || a.Items["go"] != 5 {
		t.Fatalf("nâng cấp gạch: %v thóc %d gỗ %d", err, a.Thoc, a.Items["go"])
	}
	if _, err := h.UpgradeHouse(a, "go", t0.Add(13*time.Minute), 1); err == nil {
		t.Fatal("không hạ cấp vật liệu")
	}
	h.Tick(t0.Add(2 * time.Hour))
	if regen, cap := h.comfort(); regen != 2 || cap != 16 {
		t.Fatal("nhà gạch: +2/phút, 16 khách")
	}
	old := &Home{Coop: &CoopJob{Roof: "do", State: "done"}}
	old.normalizeHouse()
	if old.Coop.Design != "chuong_nho" || old.Coop.Material != "go" || old.LotPlace != "duong" {
		t.Fatal("chuồng cũ thành Chuồng nhỏ – Gỗ cạnh đường")
	}
}

// Cấp 11–20 (PLAN_LOGIC_GAME 16): đủ quà mỗi cấp, danh hiệu cấp 10 và 20, Lô đồi mở ở cấp 12, nhà gạch hai
// tầng mở ở cấp 15 trên Lô đồi; nhà lớn chiếm nền rộng hơn và đồ trên nền về kho.
func TestLevelsElevenToTwenty(t *testing.T) {
	for lv := 2; lv <= LevelCap; lv++ {
		rw, ok := LevelRewards[lv]
		if !ok || rw.Thoc <= 0 {
			t.Fatalf("thiếu quà cấp %d", lv)
		}
		for id := range rw.Items {
			if itemByID(id) == nil {
				t.Fatalf("quà cấp %d có món lạ %s", lv, id)
			}
		}
	}
	a := newAcc()
	a.Level = 10
	a.CheckTitles()
	if !a.hasTitle("lao_lang") || a.hasTitle("truong_lang") {
		t.Fatal("cấp 10: Gà Lão Làng")
	}
	a.Level = 20
	a.CheckTitles()
	if !a.hasTitle("truong_lang") {
		t.Fatal("cấp 20: Trưởng Làng")
	}

	h := NewHome("owner123456789012345678", "Gà Chủ")
	b := newAcc()
	b.Thoc, b.Level = 50000, 11
	if _, _, err := h.BuyLot(b, 3, "doi_thong", t0); err == nil {
		t.Fatal("Lô đồi cần cấp 12")
	}
	b.Level = 12
	if _, _, err := h.BuyLot(b, 3, "doi_thong", t0); err != nil || h.Tier != 3 {
		t.Fatalf("mua Lô đồi: %v", err)
	}
	b.Items["gach"], b.Items["go"], b.Items["ngoi"], b.Items["da"] = 60, 30, 20, 10
	if _, err := h.BuildHouse(b, "nha_hai_tang", "gach", "do", t0, 1); err == nil {
		t.Fatal("nhà gạch hai tầng mở ở cấp 15")
	}
	b.Level = 15
	cx, cz := coopCell(h.tier())
	h.Objects = append(h.Objects, PlacedObj{ID: "x", Kind: "thung_go", X: cx - 2, Z: cz + 1})
	if _, err := h.BuildHouse(b, "nha_hai_tang", "gach", "do", t0, 1); err != nil {
		t.Fatalf("xây nhà hai tầng: %v", err)
	}
	if h.Storage["thung_go"] != 1 || h.cellOK(cx+2, cz+1, "") == nil {
		t.Fatal("nền nhà hai tầng rộng hơn: đồ trên nền về kho, không đặt đồ lên nền")
	}
	h.Tick(t0.Add(3 * time.Hour))
	if regen, capacity := h.comfort(); regen != 2.5 || capacity != 20 {
		t.Fatalf("nhà gạch hai tầng: hồi %.1f/phút, chứa %d", regen, capacity)
	}
}
