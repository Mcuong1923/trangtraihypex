package main

import (
	"errors"
	"fmt"
	"time"
)

// Mua đất và xây nhà (PLAN_LOGIC_GAME mục 13). Ông Chủ Đầu Tư bán lô đất ở "Văn phòng đất nền" cạnh cổng
// Xóm Rơm Vàng; Bò Mộc xây nhà theo thiết kế + vật liệu, chạy giờ cả khi offline. Nhà vẫn là phòng riêng
// của từng người nên không có chuyện hết lô.

type PlaceDef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cost int    `json:"cost"` // phụ phí vị trí: chỉ đổi cảnh quanh nhà
}

var LandPlaces = []PlaceDef{
	{"duong", "Cạnh đường làng", 0},
	{"ven_ao", "Ven ao", 300},
	{"doi_thong", "Đồi thông", 600},
}

func placeByID(id string) *PlaceDef {
	for i := range LandPlaces {
		if LandPlaces[i].ID == id {
			return &LandPlaces[i]
		}
	}
	return nil
}

type HouseRecipe struct {
	Thoc    int            `json:"thoc"`
	Items   map[string]int `json:"items"`
	Minutes int            `json:"minutes"` // thời gian Bò Mộc xây ở bộ live
}

type HouseDesign struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	MinTier  int                    `json:"minTier"`
	MinLevel int                    `json:"minLevel"`
	Recipes  map[string]HouseRecipe `json:"recipes"` // vật liệu → công thức
	// Nhà lớn ấm cúng hơn và chứa thêm khách, cộng trên độ ấm cúng của vật liệu.
	ExtraRegen float64 `json:"extraRegen"`
	ExtraCap   int     `json:"extraCap"`
}

var HouseDesigns = []HouseDesign{
	{"chuong_nho", "Chuồng nhỏ", 1, 0, map[string]HouseRecipe{
		"rom":  {200, map[string]int{"rom": 20, "go": 5}, 10},
		"go":   {400, map[string]int{"go": 20, "dinh": 10}, 20},
		"gach": {800, map[string]int{"gach": 20, "go": 10, "ngoi": 6}, 40},
	}, 0, 0},
	{"nha_hai_gian", "Nhà hai gian", 2, 0, map[string]HouseRecipe{
		"go":   {900, map[string]int{"go": 40, "dinh": 20, "rom": 10}, 40},
		"gach": {1800, map[string]int{"gach": 40, "go": 20, "ngoi": 12}, 60},
	}, 0, 2},
	// Cấp 15 (PLAN_LOGIC_GAME 16): nhà gạch hai tầng trên Lô đồi.
	{"nha_hai_tang", "Nhà gạch hai tầng", 3, 15, map[string]HouseRecipe{
		"gach": {3000, map[string]int{"gach": 60, "go": 30, "ngoi": 20, "da": 10}, 90},
	}, 0.5, 4},
}

var MaterialRank = map[string]int{"rom": 0, "go": 1, "gach": 2}
var MaterialNames = map[string]string{"rom": "rơm", "go": "gỗ", "gach": "gạch"}

// Comfort: độ ấm cúng theo vật liệu: hồi độ no khi ngủ ổ ở nhà (điểm/phút) và sức chứa khách.
var Comfort = map[string]struct {
	Regen float64 `json:"regen"`
	Cap   int     `json:"cap"`
}{"rom": {1, 8}, "go": {1.5, 12}, "gach": {2, 16}}

func designByID(id string) *HouseDesign {
	for i := range HouseDesigns {
		if HouseDesigns[i].ID == id {
			return &HouseDesigns[i]
		}
	}
	return nil
}

// buildScale: thời gian xây trong công thức tính theo bộ live (BuildDuration 10 phút); bộ khác co theo tỉ lệ.
func buildScale(b Balance) float64 { return b.BuildDuration.Seconds() / 600 }

// houseDone: nhà đã xây xong (có thể đang nâng cấp thì vẫn chưa xong).
func (h *Home) houseDone() bool { return h.Coop != nil && h.Coop.State == "done" }

// comfort trả độ hồi độ no khi ngủ ổ và sức chứa khách hiện tại.
func (h *Home) comfort() (float64, int) {
	if h.Coop == nil || h.Coop.Material == "" {
		return 0, HomeCapacity
	}
	c := Comfort[h.Coop.Material]
	regen, capacity := c.Regen, c.Cap
	if d := designByID(h.Coop.Design); d != nil {
		regen, capacity = regen+d.ExtraRegen, capacity+d.ExtraCap
	}
	if !h.houseDone() {
		return 0, max(HomeCapacity, capacity)
	}
	return regen, capacity
}

// normalizeHouse: chuồng xây trước bản cập nhật coi như "Chuồng nhỏ – Gỗ"; nhà cũ đứng cạnh đường làng.
func (h *Home) normalizeHouse() {
	if h.Coop != nil && h.Coop.Design == "" {
		h.Coop.Design, h.Coop.Material = "chuong_nho", "go"
	}
	if h.LotPlace == "" {
		h.LotPlace = "duong"
	}
}

// BuyLot mua lô đất (lên hạng) và/hoặc đổi vị trí. Đồ trang trí không còn chỗ hợp lệ về kho, không mất.
func (h *Home) BuyLot(a *Account, tier int, place string, now time.Time) (int, int, error) {
	if tier < 0 || tier >= len(LandTiers) {
		return 0, 0, errors.New("Lô đất không hợp lệ.")
	}
	pl := placeByID(place)
	if pl == nil {
		return 0, 0, errors.New("Vị trí không hợp lệ.")
	}
	if tier < h.Tier {
		return 0, 0, errors.New("Bạn đang có lô lớn hơn rồi.")
	}
	t := LandTiers[tier]
	if a.Level < t.Level {
		return 0, 0, fmt.Errorf("Cần cấp %d để mua %s.", t.Level, t.Name)
	}
	cost := 0
	if tier > h.Tier {
		cost += t.Cost
	}
	if place != h.LotPlace {
		cost += pl.Cost
	}
	if cost == 0 && tier == h.Tier && place == h.LotPlace {
		return 0, 0, errors.New("Bạn đã sở hữu đúng lô này.")
	}
	if err := a.addThoc(-cost, "buy_land", fmt.Sprintf("%s:%s", t.Name, pl.ID), now); err != nil {
		return 0, 0, err
	}
	h.Tier, h.LotPlace = tier, place
	keep, moved := h.Objects[:0:0], 0
	for _, o := range h.Objects {
		if h.cellOK(o.X, o.Z, o.ID) == nil {
			keep = append(keep, o)
		} else {
			h.Storage[o.Kind]++
			moved++
		}
	}
	h.Objects = keep
	return moved, cost, nil
}

// BuildHouse: Bò Mộc xây theo thiết kế và vật liệu; trừ thóc + vật liệu ngay, xây xong theo giờ.
func (h *Home) BuildHouse(a *Account, design, material, roof string, now time.Time, scale float64) (time.Duration, error) {
	d := designByID(design)
	if d == nil {
		return 0, errors.New("Thiết kế không hợp lệ.")
	}
	rc, ok := d.Recipes[material]
	if !ok {
		return 0, fmt.Errorf("%s không xây bằng %s.", d.Name, MaterialNames[material])
	}
	if roof == "" && h.Coop != nil {
		roof = h.Coop.Roof
	}
	if _, ok := RoofColors[roof]; !ok {
		return 0, errors.New("Chọn màu mái hợp lệ.")
	}
	switch {
	case a.Level < d.MinLevel:
		return 0, fmt.Errorf("%s mở ở cấp %d.", d.Name, d.MinLevel)
	case h.Tier < d.MinTier:
		return 0, fmt.Errorf("%s cần %s trở lên.", d.Name, LandTiers[d.MinTier].Name)
	case h.Coop != nil && h.Coop.State == "building":
		return 0, errors.New("Bò Mộc đang xây dở, chờ xong đã.")
	case h.Coop != nil && h.Coop.Design == design:
		return 0, errors.New("Nhà này đã xây; muốn chắc hơn thì nâng cấp vật liệu.")
	}
	if a.Thoc < rc.Thoc {
		return 0, fmt.Errorf("Cần %d thóc tiền công Bò Mộc.", rc.Thoc)
	}
	if err := a.takeItems(rc.Items); err != nil {
		return 0, err
	}
	_ = a.addThoc(-rc.Thoc, "build_house", design+":"+material, now)
	dur := time.Duration(float64(rc.Minutes) * float64(time.Minute) * scale)
	h.Coop = &CoopJob{Roof: roof, Design: design, Material: material, State: "building", StartAt: now, DoneAt: now.Add(dur)}
	h.clearFoot()
	return dur, nil
}

// UpgradeHouse đổi sang vật liệu tốt hơn: chỉ trả phần vật liệu chênh lệch và một nửa tiền công.
func (h *Home) UpgradeHouse(a *Account, material string, now time.Time, scale float64) (time.Duration, error) {
	if !h.houseDone() {
		return 0, errors.New("Cần một ngôi nhà đã xây xong để nâng cấp.")
	}
	d := designByID(h.Coop.Design)
	cur, next := d.Recipes[h.Coop.Material], d.Recipes[material]
	if _, ok := d.Recipes[material]; !ok || MaterialRank[material] <= MaterialRank[h.Coop.Material] {
		return 0, errors.New("Chọn vật liệu tốt hơn vật liệu hiện tại.")
	}
	diff := map[string]int{}
	for id, n := range next.Items {
		if extra := n - cur.Items[id]; extra > 0 {
			diff[id] = extra
		}
	}
	fee := next.Thoc / 2
	if a.Thoc < fee {
		return 0, fmt.Errorf("Cần %d thóc tiền công nâng cấp.", fee)
	}
	if err := a.takeItems(diff); err != nil {
		return 0, err
	}
	_ = a.addThoc(-fee, "upgrade_house", material, now)
	dur := time.Duration(float64(next.Minutes) * float64(time.Minute) * scale / 2)
	h.Coop = &CoopJob{Roof: h.Coop.Roof, Design: h.Coop.Design, Material: material, State: "building", StartAt: now, DoneAt: now.Add(dur)}
	return dur, nil
}

// withHome sửa nhà của owner an toàn: nhà đang mở phòng thì sửa trong goroutine của phòng đó; chưa mở thì
// sửa dưới khoá hub (không phòng nào mở được nhà giữa chừng) rồi lưu.
func (h *Hub) withHome(owner, name string, f func(*Home) error) error {
	h.mu.Lock()
	room := h.homes[owner]
	if room == nil {
		defer h.mu.Unlock()
		home, err := h.store.Homes.Get(owner, name)
		if err != nil {
			return err
		}
		home.normalizeHouse()
		if err := f(home); err != nil {
			return err
		}
		return h.store.Homes.Save(home)
	}
	h.mu.Unlock()
	errc := make(chan error, 1)
	room.Do(func() {
		room.home.normalizeHouse()
		err := f(room.home)
		if err == nil {
			err = h.store.Homes.Save(room.home)
			room.broadcastHome()
		}
		errc <- err
	})
	select {
	case err := <-errc:
		return err
	case <-time.After(3 * time.Second):
		return errors.New("Nhà đang bận, thử lại sau.")
	}
}
