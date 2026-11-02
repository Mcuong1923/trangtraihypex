package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Hạt giống và trồng cây (PLAN_LOGIC_GAME mục 14). Luống ở Ao và vườn (12 luống, mỗi người giữ tối đa 1)
// và luống rau trong nhà riêng. Cây là tài sản riêng của từng người (lưu trong tài khoản), lớn theo giờ thực
// cả khi offline, nên không ai hái được cây của người khác.

type Plant struct {
	Seed    string    `json:"seed"`
	At      time.Time `json:"at"`
	Ready   time.Time `json:"ready"`
	Wilt    time.Time `json:"wilt"`
	Watered bool      `json:"watered"`
	Fertile bool      `json:"fertile"` // đã bón phân ủ: thêm 1 nông sản
}

type SeedDef struct {
	Seed  string        `json:"seed"`
	Crop  string        `json:"crop"`
	Grow  time.Duration `json:"-"`
	Sec   float64       `json:"sec"`
	Yield int           `json:"yield"`
}

var Seeds = []SeedDef{
	{"hat_cai", "cai", 10 * time.Minute, 600, 3},
	{"hat_ca_rot", "ca_rot", 20 * time.Minute, 1200, 3},
	{"hat_bi_ngo", "bi_ngo", 45 * time.Minute, 2700, 1},
	{"hat_hoa", "bo_hoa", 30 * time.Minute, 1800, 1},
}

const (
	ParkBedHold = 2 * time.Hour // luống cộng đồng giữ tối đa 2 giờ
	WaterCut    = 0.25          // tưới: rút 25% thời gian còn lại, mỗi lần trồng tưới một lần
)

func seedByID(id string) *SeedDef {
	for i := range Seeds {
		if Seeds[i].Seed == id {
			return &Seeds[i]
		}
	}
	return nil
}

func isParkBed(key string) bool { return strings.HasPrefix(key, "park:") }

// PlantSeed gieo hạt vào luống. growScale < 1 chỉ dùng cho bộ cân bằng thử nhanh.
func (a *Account) PlantSeed(bed, seed string, fertilize bool, now time.Time, growScale float64) (*Plant, error) {
	sd := seedByID(seed)
	switch {
	case sd == nil:
		return nil, errors.New("Hạt giống không hợp lệ.")
	case a.Items[seed] <= 0:
		return nil, errors.New("Bạn không còn " + itemByID(seed).Name + ". Mua ở Quầy thức ăn.")
	case a.Beds[bed] != nil:
		return nil, errors.New("Luống này đang có cây.")
	case fertilize && a.Items["phan_u"] <= 0:
		return nil, errors.New("Bạn chưa có phân ủ.")
	}
	if isParkBed(bed) {
		for k := range a.Beds {
			if isParkBed(k) {
				return nil, errors.New("Mỗi người chỉ giữ một luống cộng đồng. Thu hoạch luống cũ trước.")
			}
		}
	}
	grow := time.Duration(float64(sd.Grow) * growScale)
	wilt := 3 * grow
	if isParkBed(bed) && wilt > ParkBedHold {
		wilt = ParkBedHold
	}
	a.Items[seed]--
	if fertilize {
		a.Items["phan_u"]--
	}
	pl := &Plant{Seed: seed, At: now, Ready: now.Add(grow), Wilt: now.Add(max(wilt, grow+time.Minute)), Fertile: fertilize}
	a.Beds[bed] = pl
	return pl, nil
}

func (a *Account) WaterBed(bed string, now time.Time) error {
	pl := a.Beds[bed]
	switch {
	case pl == nil:
		return errors.New("Luống này chưa có cây của bạn.")
	case pl.Watered:
		return errors.New("Đã tưới rồi, để cây nghỉ.")
	case !now.Before(pl.Ready):
		return errors.New("Cây chín rồi, thu hoạch thôi!")
	}
	pl.Watered = true
	pl.Ready = pl.Ready.Add(-time.Duration(float64(pl.Ready.Sub(now)) * WaterCut))
	return nil
}

// Harvest thu cây chín; để quá hạn thì cây héo, chỉ còn 1 phân ủ.
func (a *Account) Harvest(bed string, now time.Time) (string, int, error) {
	pl := a.Beds[bed]
	if pl == nil {
		return "", 0, errors.New("Luống này chưa có cây của bạn.")
	}
	if now.Before(pl.Ready) {
		return "", 0, fmt.Errorf("Cây chưa chín, còn %s.", humanDur(pl.Ready.Sub(now)))
	}
	sd := seedByID(pl.Seed)
	crop, n := sd.Crop, sd.Yield
	if pl.Fertile {
		n++
	}
	if now.After(pl.Wilt) {
		crop, n = "phan_u", 1
	}
	if err := a.addItems(map[string]int{crop: n}); err != nil {
		return "", 0, err
	}
	delete(a.Beds, bed)
	return crop, n, nil
}
