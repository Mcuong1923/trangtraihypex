package main

import (
	"math"
	"testing"
	"time"
)

// Luật V1.1/V1.2 dạng hàm thuần.

func TestRegionalWormTable(t *testing.T) {
	sum, ev := 0.0, 0.0
	for _, w := range VenSuoiWorms {
		sum += w.Prob
		ev += w.Prob * float64(w.Sell)
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("bảng Ven Suối tổng %v", sum)
	}
	// Mô phỏng lại thu nhập sau tiền vé (mục 7.2.1): 20 lượt − vé khứ hồi vẫn phải đáng đi.
	base := 0.0
	for _, w := range Worms {
		base += w.Prob * float64(w.Sell)
	}
	if ev*20-2*BusFare <= base*20 {
		t.Fatalf("Ven Suối %.0f − vé %d phải hơn trại gốc %.0f", ev*20, 2*BusFare, base*20)
	}
	if RollWormIn(RegionVenSuoi, 0.1).ID != "suoi" || RollWormIn(RegionVenSuoi, 0.9999).ID != "rong" || RollWorm(0.1).ID != "dat" {
		t.Fatal("chọn đúng bảng theo vùng")
	}
}

func TestIncubationAndCompanions(t *testing.T) {
	b := PlanBalance()
	a := newAcc()
	if a.ClaimEggs(t0) != 0 {
		t.Fatal("cấp 1 chưa có trứng")
	}
	a.Level = 7
	// Quà cấp 2..7 có trứng ở cấp 4, 5, 7; nhận một lần.
	if n := a.ClaimEggs(t0); n != 3 || a.ClaimEggs(t0) != 0 || a.Items["trung_thuong"] != 3 {
		t.Fatalf("quà cấp phải cho 3 trứng một lần, có %d", n)
	}
	inc, err := a.StartIncubation("", t0, b)
	if err != nil || a.Items["trung_thuong"] != 2 || inc.Egg != "trung_thuong" {
		t.Fatal("bắt đầu ấp tiêu đúng một trứng")
	}
	if _, err := a.StartIncubation("", t0, b); err == nil {
		t.Fatal("một ổ một trứng")
	}
	if _, err := a.Hatch(inc.ID, "Bông Con", 0.3, 0.3, t0.Add(time.Hour)); err == nil {
		t.Fatal("chưa đủ 2 giờ")
	}
	c, err := a.Hatch(inc.ID, "Bông Con", 0.3, 0.3, t0.Add(2*time.Hour))
	if err != nil || len(a.Companions) != 1 || a.Following != c.ID || c.Species != "ga_con" {
		t.Fatalf("nở gà con: %v %+v", err, c)
	}
	if _, err := a.Hatch(inc.ID, "Bông Con", 0.3, 0.3, t0.Add(3*time.Hour)); err == nil || len(a.Companions) != 1 {
		t.Fatal("gửi lại yêu cầu nở không tạo thêm gà con")
	}
	if g, err := a.Care(c.ID, "pet", t0); err != nil || g != 5 {
		t.Fatal("vuốt ve")
	}
	if _, err := a.Care(c.ID, "pet", t0); err == nil {
		t.Fatal("vuốt ve 1 lần/ngày")
	}
	a.Food = map[string]int{} // quà cấp có cám và rau; làm trống túi để thử trường hợp không có gì ăn
	if _, err := a.Care(c.ID, "feed", t0); err == nil {
		t.Fatal("cho ăn cần thức ăn trong túi")
	}
	a.Food["cam"] = 5
	for i := 0; i < 3; i++ {
		a.Care(c.ID, "feed", t0)
	}
	if _, err := a.Care(c.ID, "feed", t0); err == nil || a.Food["cam"] != 2 || a.Companions[0].Affinity != 35 {
		t.Fatalf("cho ăn tối đa 3 lần/ngày: thân thiết %d", a.Companions[0].Affinity)
	}
	if _, err := a.Care(c.ID, "pet", t0.Add(24*time.Hour)); err != nil {
		t.Fatal("ngày mới chăm lại được")
	}
}

func TestBookingTokens(t *testing.T) {
	bal := PlanBalance()
	a := newAcc()
	if _, err := a.Book(RegionHypex, RegionVenSuoi, t0, bal); err == nil {
		t.Fatal("cần cấp 3 để tới Ven Suối")
	}
	a.Level = 3
	for i := 0; i < MaxBusTokens; i++ {
		bk, err := a.Book(RegionHypex, RegionVenSuoi, t0, bal)
		if err != nil || !bk.Token {
			t.Fatalf("lượt miễn phí %d: %v", i, err)
		}
		a.Booking = nil // giả lập đã đi xong
	}
	if _, err := a.Book(RegionHypex, RegionVenSuoi, t0, bal); err == nil {
		t.Fatal("hết lượt và không có thóc thì không đặt được")
	}
	a.Thoc = 15
	bk, err := a.Book(RegionHypex, RegionVenSuoi, t0, bal)
	if err != nil || bk.Token || a.Thoc != 5 {
		t.Fatal("hết lượt thì trả 10 thóc")
	}
	if _, err := a.Book(RegionHypex, RegionVenSuoi, t0, bal); err == nil {
		t.Fatal("một người một vé đang giữ")
	}
	if _, err := a.CancelBooking(t0, bal); err != nil || a.Thoc != 15 {
		t.Fatal("huỷ vé trả tiền hoàn thóc")
	}
	if _, err := a.CancelBooking(t0, bal); err == nil || a.Thoc != 15 {
		t.Fatal("hoàn đúng một lần")
	}
	// Hồi 1 lượt mỗi 5 phút, tối đa 2.
	if a.Tokens(t0.Add(4*time.Minute), bal) != 0 || a.Tokens(t0.Add(5*time.Minute+time.Second), bal) != 1 {
		t.Fatal("5 phút hồi 1 lượt")
	}
	if a.Tokens(t0.Add(time.Hour), bal) != MaxBusTokens {
		t.Fatal("tối đa 2 lượt")
	}
	// Huỷ vé dùng lượt thì trả lượt; về Hypex luôn miễn phí.
	bk, _ = a.Book(RegionHypex, RegionVenSuoi, t0.Add(time.Hour), bal)
	a.CancelBooking(t0.Add(time.Hour), bal)
	if !bk.Token || a.Tokens(t0.Add(time.Hour), bal) != MaxBusTokens {
		t.Fatal("huỷ vé trả lại lượt")
	}
	home, err := a.Book(RegionVenSuoi, RegionHypex, t0.Add(time.Hour), bal)
	if err != nil || !home.Free || home.Token || home.Paid != 0 {
		t.Fatal("về Hypex miễn phí, không tiêu lượt")
	}
	a.Booking = nil
	// Vé đã chạy thì không huỷ.
	a.Book(RegionHypex, RegionVenSuoi, t0.Add(time.Hour), bal)
	a.Booking.State = "departed"
	if _, err := a.CancelBooking(t0.Add(time.Hour), bal); err == nil {
		t.Fatal("xe đã chạy không huỷ")
	}
}

func TestVehicleShop(t *testing.T) {
	a := newAcc()
	a.Thoc = 20000
	if err := a.BuyVehicle("xe_may", t0); err == nil {
		t.Fatal("xe máy cần cấp 5")
	}
	if err := a.BuyVehicle("xe_dap", t0); err != nil || a.Thoc != 19200 {
		t.Fatal("mua xe đạp 800")
	}
	if err := a.BuyVehicle("xe_dap", t0); err == nil || a.Thoc != 19200 {
		t.Fatal("không mua trùng")
	}
}

func TestContractStoreRules(t *testing.T) {
	cs, _ := OpenContracts(t.TempDir())
	k, err := cs.Create("owner", "Chủ", "do", 300, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.Create("owner", "Chủ", "do", 300, 2, time.Hour); err == nil {
		t.Fatal("một chủ một hợp đồng đang mở")
	}
	if _, err := cs.Take(k.ID, "owner", "Chủ"); err == nil {
		t.Fatal("không tự nhận hợp đồng của mình")
	}
	if _, err := cs.Take(k.ID, "tho", "Thợ"); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.Cancel(k.ID, "owner"); err == nil {
		t.Fatal("thợ đã nhận thì chủ không huỷ ngang")
	}
	now := time.Now()
	// Hồi quy: thi công ở nhà người khác (vd nhà của chính thợ) bị từ chối và KHÔNG cộng tiến độ.
	for i := 0; i < 3; i++ {
		if _, err := cs.Work(k.ID, "tho", "tho", now.Add(time.Duration(i-10)*2*time.Second)); err == nil {
			t.Fatal("thi công sai nhà phải bị chặn")
		}
	}
	if got := cs.Contracts[k.ID]; got.Progress != 0 || got.State != "taken" || len(cs.Credits["tho"]) != 0 {
		t.Fatalf("thi công sai nhà không được cộng tiến độ hay trả công: progress=%d state=%s", got.Progress, got.State)
	}
	cs.Work(k.ID, "tho", "owner", now)
	if _, err := cs.Work(k.ID, "tho", "owner", now.Add(100*time.Millisecond)); err == nil {
		t.Fatal("thi công dồn dập phải bị chặn")
	}
	done, err := cs.Work(k.ID, "tho", "owner", now.Add(2*time.Second))
	if err != nil || done.State != "done" {
		t.Fatalf("đủ tiến độ thì nghiệm thu: %v", err)
	}
	if c := cs.TakeCredits("tho"); len(c) != 1 || c[0].Amount != 300 || len(cs.TakeCredits("tho")) != 0 {
		t.Fatal("trả công đúng một lần")
	}
	if _, err := cs.Work(k.ID, "tho", "owner", now.Add(5*time.Second)); err == nil {
		t.Fatal("hợp đồng xong không thi công tiếp")
	}
	// Quá hạn: hoàn chủ đúng một lần.
	k2, _ := cs.Create("owner", "Chủ", "xanh", 500, 2, time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if ex := cs.Expire(time.Now()); len(ex) != 1 || ex[0].ID != k2.ID || len(cs.Expire(time.Now())) != 0 {
		t.Fatal("quá hạn xử lý một lần")
	}
	if c := cs.TakeCredits("owner"); len(c) != 1 || c[0].Amount != 500 {
		t.Fatal("hoàn ký quỹ cho chủ")
	}
}

func TestGuildStoreRules(t *testing.T) {
	gs, _ := OpenGuilds(t.TempDir())
	gd, err := gs.Create("o1", "Chủ", "Hội Thóc Vàng", "🌾", "#3f86c9")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gs.Create("o2", "Khác", "hội thóc vàng", "🐔", ""); err == nil {
		t.Fatal("tên hội trùng (không phân biệt hoa thường)")
	}
	if r, _ := gs.Join("m1", "Thành viên", gd.ID); r != "requested" {
		t.Fatal("hội mặc định cần duyệt")
	}
	gs.Do("o1", func(g *Guild) error { return gs.addMemberLocked(g, "m1", "Thành viên") })
	if err := gs.Leave("o1"); err == nil {
		t.Fatal("chủ hội phải chuyển quyền trước khi rời")
	}
	gs.AddFund("m1", 100)
	if v := gs.View("o1"); v.Fund != 100 || v.Members["m1"].Contributed != 100 {
		t.Fatal("sổ đóng góp")
	}
	for i := 0; i < GuildMax; i++ {
		gs.Do("o1", func(g *Guild) error { return gs.addMemberLocked(g, "x"+string(rune('a'+i)), "X") })
	}
	if v := gs.View("o1"); len(v.Members) != GuildMax {
		t.Fatalf("tối đa %d thành viên, có %d", GuildMax, len(v.Members))
	}
}

func TestSeasonAndParty(t *testing.T) {
	autumn := time.Date(2026, 10, 31, 12, 0, 0, 0, vnZone)
	spring := time.Date(2026, 5, 1, 12, 0, 0, 0, vnZone)
	if RollWormAt(RegionHypex, 0.001, autumn).ID != "bi_ngo" || RollWormAt(RegionHypex, 0.001, spring).ID == "bi_ngo" {
		t.Fatal("giun bí ngô chỉ có trong mùa thu")
	}
	// Phân bố sau khi co lại vẫn giữ thứ tự các khoảng: r gần 1 vẫn ra giun hiếm nhất.
	if RollWormAt(RegionHypex, 0.9995, autumn).ID != "cauvong" || RollWormAt(RegionHypex, 0.5, autumn).ID != "dat" {
		t.Fatal("bảng thường co lại theo tỉ lệ")
	}
	a := newAcc()
	a.Thoc = 2000
	if err := a.BuyCosmetic("mu_bi_ngo", spring); err == nil {
		t.Fatal("đồ mùa thu không bán ngoài mùa")
	}
	if err := a.BuyCosmetic("mu_bi_ngo", autumn); err != nil {
		t.Fatal(err)
	}
	h := NewHome("o", "Chủ")
	if err := h.OpenParty(30, autumn); err == nil {
		t.Fatal("mở tiệc cần bàn tiệc")
	}
	h.Objects = append(h.Objects, PlacedObj{ID: "t", Kind: "ban_tiec", X: -2, Z: 0})
	if err := h.OpenParty(45, autumn); err == nil {
		t.Fatal("chỉ 30 hoặc 60 phút")
	}
	if h.OpenParty(30, autumn) != nil || !h.Partying(autumn.Add(29*time.Minute)) || h.Partying(autumn.Add(31*time.Minute)) {
		t.Fatal("tiệc có giới hạn thời gian")
	}
}
