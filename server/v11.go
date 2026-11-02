package main

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Luật V1.1: ấp trứng & bạn đồng hành (mục 12), vé xe buýt & lượt miễn phí (mục 8.5), mua xe (mục 8.4).

// ---------- Ấp trứng & bạn đồng hành ----------

type Incubation struct {
	ID       string    `json:"id"`
	Egg      string    `json:"egg"` // loại trứng đang ấp: trung_thuong | trung_dom | trung_vang
	StartAt  time.Time `json:"startAt"`
	DoneAt   time.Time `json:"doneAt"`
	Notified bool      `json:"notified"`
}

type Companion struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Species   string    `json:"species"`
	Color     string    `json:"color"`
	Affinity  int       `json:"affinity"`
	HatchedAt time.Time `json:"hatchedAt"`
	CareDay   string    `json:"careDay"`
	Petted    bool      `json:"petted"`
	Fed       int       `json:"fed"`
}

var ChickColors = []string{"#f4cf3f", "#f5f2e9", "#3b3530", "#b5743c", "#e9933a"}

// Độ thân thiết mở biểu cảm/phụ kiện cho bạn đồng hành (mục 12).
func AffinityLevel(n int) (int, string) {
	switch {
	case n >= 150:
		return 2, "Thân thiết"
	case n >= 50:
		return 1, "Thân"
	}
	return 0, "Mới quen"
}

// ClaimEggs nhận các phần quà lên cấp còn chờ (trứng nằm trong bảng quà cấp, PLAN_LOGIC_GAME 12); trả số trứng mới.
func (a *Account) ClaimEggs(now time.Time) int {
	before := a.eggCount()
	a.ClaimLevelRewards(now)
	return a.eggCount() - before
}

// EggSpecies: loài có thể nở theo loại trứng (PLAN_LOGIC_GAME 15.1); tổng mỗi dòng bằng 1.
var EggSpecies = map[string][]struct {
	Species string
	P       float64
}{
	"trung_thuong": {{"ga_con", .60}, {"vit_con", .25}, {"ngong_con", .15}},
	"trung_dom":    {{"chim_cut", .35}, {"ga_tre", .35}, {"ga_ac", .30}},
	"trung_vang":   {{"cong_con", .50}, {"ga_con", .0834}, {"vit_con", .0833}, {"ngong_con", .0833}, {"chim_cut", .0833}, {"ga_tre", .0833}, {"ga_ac", .0834}},
}

var EggKinds = []string{"trung_thuong", "trung_dom", "trung_vang"}

// CompanionCap: 3 con mang theo; nhà đã xây xong có chuồng nuôi thêm 6 con.
func (a *Account) CompanionCap() int {
	if a.HouseBuilt {
		return CompanionMax + 6
	}
	return CompanionMax
}

// StartIncubation tiêu đúng một trứng (loại kind, rỗng = loại đầu tiên đang có); ấp một trứng một lúc.
func (a *Account) StartIncubation(kind string, now time.Time, b Balance) (*Incubation, error) {
	if a.Incubation != nil {
		return nil, errors.New("Ổ ấp đang có trứng.")
	}
	if kind == "" {
		for _, k := range EggKinds {
			if a.Items[k] > 0 {
				kind = k
				break
			}
		}
	}
	if !contains(EggKinds, kind) || a.Items[kind] <= 0 {
		return nil, errors.New("Bạn chưa có trứng. Trứng có trong quà lên cấp, thưởng hướng dẫn và điểm danh.")
	}
	if len(a.Companions) >= a.CompanionCap() {
		return nil, fmt.Errorf("Đã đủ %d bạn đồng hành. Xây nhà để có chuồng nuôi thêm.", a.CompanionCap())
	}
	a.Items[kind]--
	a.Incubation = &Incubation{ID: a.nextID("egg"), Egg: kind, StartAt: now, DoneAt: now.Add(b.HatchDuration)}
	return a.Incubation, nil
}

// hatchSpecies chọn loài theo bảng của loại trứng với số ngẫu nhiên r ∈ [0,1).
func hatchSpecies(egg string, r float64) string {
	tab := EggSpecies[egg]
	if len(tab) == 0 {
		tab = EggSpecies["trung_thuong"]
	}
	acc := 0.0
	for _, e := range tab {
		acc += e.P
		if r < acc {
			return e.Species
		}
	}
	return tab[len(tab)-1].Species
}

// Hatch tạo đúng một gà con cho đúng một lần ấp; gửi lại yêu cầu không tạo thêm (mục 12).
func (a *Account) Hatch(jobID, name string, colorR, speciesR float64, now time.Time) (Companion, error) {
	inc := a.Incubation
	if inc == nil || inc.ID != jobID {
		return Companion{}, errors.New("Không có trứng nào đang chờ nở.")
	}
	if now.Before(inc.DoneAt) {
		return Companion{}, fmt.Errorf("Trứng còn %.0f giây nữa mới nở.", math.Ceil(inc.DoneAt.Sub(now).Seconds()))
	}
	n, err := ValidateName(name)
	if err != nil {
		return Companion{}, err
	}
	sp := hatchSpecies(inc.Egg, speciesR)
	color := speciesByID(sp).Body
	if sp == "ga_con" {
		color = ChickColors[int(colorR*float64(len(ChickColors)))%len(ChickColors)]
	}
	c := Companion{ID: a.nextID("pet"), Name: n, Species: sp, Color: color, HatchedAt: now}
	a.Companions = append(a.Companions, c)
	a.Incubation = nil
	if a.Following == "" {
		a.Following = c.ID
	}
	return c, nil
}

func (a *Account) companion(id string) *Companion {
	for i := range a.Companions {
		if a.Companions[i].ID == id {
			return &a.Companions[i]
		}
	}
	return nil
}

func (a *Account) Follow(id string) error {
	if id != "" && a.companion(id) == nil {
		return errors.New("Không tìm thấy bạn đồng hành.")
	}
	a.Following = id
	return nil
}

func (a *Account) RenameCompanion(id, name string) error {
	c := a.companion(id)
	if c == nil {
		return errors.New("Không tìm thấy bạn đồng hành.")
	}
	n, err := ValidateName(name)
	if err != nil {
		return err
	}
	c.Name = n
	return nil
}

// Care: vuốt ve 1 lần/ngày (+5), cho ăn tối đa 3 lần/ngày bằng thức ăn trong túi (+10). Không có hình phạt bỏ bê.
func (a *Account) Care(id, kind string, now time.Time) (int, error) {
	c := a.companion(id)
	if c == nil {
		return 0, errors.New("Không tìm thấy bạn đồng hành.")
	}
	day := DayKey(now)
	if c.CareDay != day {
		c.CareDay, c.Petted, c.Fed = day, false, 0
	}
	switch kind {
	case "pet":
		if c.Petted {
			return 0, errors.New("Hôm nay đã vuốt ve rồi.")
		}
		c.Petted = true
		c.Affinity += 5
		return 5, nil
	case "feed":
		if c.Fed >= 3 {
			return 0, errors.New("Hôm nay đã cho ăn đủ 3 lần.")
		}
		var used string
		for _, f := range Foods {
			if a.Food[f.ID] > 0 {
				used = f.ID
				break
			}
		}
		if used == "" {
			return 0, errors.New("Cần một món thức ăn trong túi (mua ở Quầy thức ăn).")
		}
		a.Food[used]--
		c.Fed++
		c.Affinity += 10
		return 10, nil
	}
	return 0, errors.New("Cách chăm không hợp lệ.")
}

// ---------- Vé xe buýt & lượt miễn phí ----------

type Booking struct {
	ID    string    `json:"id"`
	From  string    `json:"from"`
	To    string    `json:"to"`
	Day   string    `json:"day"` // ngày quota của lượt miễn phí
	Free  bool      `json:"free"`
	Token bool      `json:"token,omitempty"` // đã tiêu một lượt miễn phí
	Paid  int       `json:"paid"`
	State string    `json:"state"` // held | departed
	Trip  string    `json:"trip,omitempty"`
	At    time.Time `json:"at"`
}

func (a *Account) ensureTravelDay(now time.Time) {
	if d := DayKey(now); a.TravelDay != d {
		a.TravelDay, a.TripsUsed = d, 0
	}
}

// TripsLeft giữ tên cũ cho client: nay là số lượt xe miễn phí đang có (PLAN_LOGIC_GAME 7.2).
func (a *Account) TripsLeft(now time.Time, b Balance) int { return a.Tokens(now, b) }

// Book giữ ghế xe liên trại: về Hypex luôn miễn phí; còn lại ưu tiên lượt miễn phí, hết thì trả BusFare
// (đã áp "Mặc Cả").
func (a *Account) Book(from, to string, now time.Time, b Balance) (*Booking, error) {
	r := Regions[to]
	if r == nil || to == from {
		return nil, errors.New("Tuyến không hợp lệ.")
	}
	if a.Booking != nil {
		return nil, errors.New("Bạn đã có vé, huỷ vé cũ trước.")
	}
	if a.Level < r.Level {
		return nil, fmt.Errorf("Cần cấp %d để tới %s.", r.Level, r.Title)
	}
	bk := &Booking{ID: a.nextID("bk"), From: from, To: to, Day: DayKey(now), State: "held", At: now}
	if to == RegionHypex {
		bk.Free = true // không bao giờ kẹt ở vùng khác
	} else {
		token, paid, err := a.payRide(BusFare, "bus_fare", bk.ID, now, b)
		if err != nil {
			return nil, fmt.Errorf("Hết lượt miễn phí (lượt mới sau %s); vé %d thóc.", humanDur(time.Duration(a.NextTokenIn(now, b))*time.Second), fareAfter(a, BusFare))
		}
		bk.Free, bk.Token, bk.Paid = token, token, paid
	}
	a.Booking = bk
	return bk, nil
}

// CancelBooking hoàn đúng một lần: lượt miễn phí về ví lượt, thóc về ví.
func (a *Account) CancelBooking(now time.Time, b Balance) (string, error) {
	bk := a.Booking
	if bk == nil {
		return "", errors.New("Bạn không có vé nào.")
	}
	if bk.State != "held" {
		return "", errors.New("Xe đã chạy, không huỷ được.")
	}
	a.Booking = nil
	switch {
	case bk.Token:
		a.refundToken(now, b)
		return "Đã huỷ vé, trả lại lượt miễn phí.", nil
	case bk.Paid > 0:
		_ = a.addThoc(bk.Paid, "bus_refund", bk.ID, now)
		return fmt.Sprintf("Đã huỷ vé, hoàn %d thóc.", bk.Paid), nil
	}
	return "Đã huỷ vé.", nil
}

// ---------- Xe ----------

func (a *Account) BuyVehicle(id string, now time.Time) error {
	v := vehicleByID(id)
	if v == nil {
		return errors.New("Loại xe không hợp lệ.")
	}
	if a.OwnedVehicles[id] {
		return errors.New("Bạn đã có xe này.")
	}
	if a.Level < v.Level {
		return fmt.Errorf("Cần cấp %d để mua %s.", v.Level, v.Name)
	}
	cost := v.Cost
	if len(a.OwnedVehicles) == 0 && a.couponLive(CouponFirstBike, now) {
		cost = v.Cost * 9 / 10 // phiếu Dê Thợ Máy: giảm 10% chiếc xe đầu tiên
	}
	if err := a.addThoc(-cost, "buy_vehicle", id, now); err != nil {
		return err
	}
	if cost != v.Cost {
		delete(a.Coupons, CouponFirstBike)
		a.Coupons[CouponFirstBike+"_used"] = now
	}
	a.OwnedVehicles[id] = true
	return nil
}

// VehiclePrice là giá xe sau phiếu giảm giá (để client hiện đúng).
func (a *Account) VehiclePrice(id string, now time.Time) int {
	v := vehicleByID(id)
	if v == nil {
		return 0
	}
	if len(a.OwnedVehicles) == 0 && a.couponLive(CouponFirstBike, now) {
		return v.Cost * 9 / 10
	}
	return v.Cost
}

func (a *Account) hatchIn(now time.Time) float64 {
	if a.Incubation == nil {
		return 0
	}
	return math.Max(0, a.Incubation.DoneAt.Sub(now).Seconds())
}
