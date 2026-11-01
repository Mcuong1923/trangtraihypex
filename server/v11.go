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

// ClaimEggs trao trứng ở các mốc cấp đã đạt nhưng chưa nhận; mỗi mốc một lần.
func (a *Account) ClaimEggs() int {
	got := 0
	for _, lv := range EggMilestones {
		if a.Level >= lv && !a.EggsClaimed[lv] {
			a.EggsClaimed[lv] = true
			a.Eggs++
			got++
		}
	}
	return got
}

// StartIncubation tiêu đúng một trứng; mỗi người ấp một trứng một lúc.
func (a *Account) StartIncubation(now time.Time, b Balance) (*Incubation, error) {
	if a.Incubation != nil {
		return nil, errors.New("Ổ ấp đang có trứng.")
	}
	if a.Eggs <= 0 {
		return nil, errors.New("Bạn chưa có trứng. Vịt Bông tặng trứng ở cấp 5, 7 và 9.")
	}
	if len(a.Companions) >= CompanionMax {
		return nil, fmt.Errorf("Đã đủ %d bạn đồng hành.", CompanionMax)
	}
	a.Eggs--
	a.Incubation = &Incubation{ID: a.nextID("egg"), StartAt: now, DoneAt: now.Add(b.HatchDuration)}
	return a.Incubation, nil
}

// Hatch tạo đúng một gà con cho đúng một lần ấp; gửi lại yêu cầu không tạo thêm (mục 12).
func (a *Account) Hatch(jobID, name string, colorR float64, now time.Time) (Companion, error) {
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
	c := Companion{ID: a.nextID("pet"), Name: n, Species: "ga_con", Color: ChickColors[int(colorR*float64(len(ChickColors)))%len(ChickColors)], HatchedAt: now}
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

func (a *Account) TripsLeft(now time.Time) int {
	a.ensureTravelDay(now)
	return max(0, FreeTrips-a.TripsUsed)
}

// Book giữ chỗ: ưu tiên dùng lượt miễn phí, hết lượt thì giữ thóc (mục 8.5).
func (a *Account) Book(from, to string, now time.Time) (*Booking, error) {
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
	a.ensureTravelDay(now)
	bk := &Booking{ID: a.nextID("bk"), From: from, To: to, Day: a.TravelDay, State: "held", At: now}
	if a.TripsUsed < FreeTrips {
		a.TripsUsed++
		bk.Free = true
	} else {
		if err := a.addThoc(-BusFare, "bus_fare", bk.ID, now); err != nil {
			return nil, fmt.Errorf("Hết lượt miễn phí hôm nay; vé %d thóc.", BusFare)
		}
		bk.Paid = BusFare
	}
	a.Booking = bk
	return bk, nil
}

// CancelBooking hoàn đúng một lần: lượt miễn phí về quota của ngày đặt, thóc về ví.
func (a *Account) CancelBooking(now time.Time) (string, error) {
	bk := a.Booking
	if bk == nil {
		return "", errors.New("Bạn không có vé nào.")
	}
	if bk.State != "held" {
		return "", errors.New("Xe đã chạy, không huỷ được.")
	}
	a.Booking = nil
	if bk.Free {
		a.ensureTravelDay(now)
		if bk.Day == a.TravelDay && a.TripsUsed > 0 {
			a.TripsUsed-- // vé đặt hôm nay: trả lượt; vé hôm trước: quota ngày cũ đã qua, không cộng lượt cho ngày mới
		}
		return "Đã huỷ vé, trả lại lượt miễn phí.", nil
	}
	_ = a.addThoc(bk.Paid, "bus_refund", bk.ID, now)
	return fmt.Sprintf("Đã huỷ vé, hoàn %d thóc.", bk.Paid), nil
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
	if err := a.addThoc(-v.Cost, "buy_vehicle", id, now); err != nil {
		return err
	}
	a.OwnedVehicles[id] = true
	return nil
}

func (a *Account) hatchIn(now time.Time) float64 {
	if a.Incubation == nil {
		return 0
	}
	return math.Max(0, a.Incubation.DoneAt.Sub(now).Seconds())
}
