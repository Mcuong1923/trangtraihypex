package main

import "time"

// Bạn đồng hành có kỹ năng (PLAN_LOGIC_GAME mục 15). Chỉ con đang đi theo mới có tác dụng; mức kỹ năng
// theo độ thân thiết (AffinityLevel 0/1/2 = Lv1/Lv2/Lv3).

const (
	AbilitySniff  = "sniff"  // Mỏ Thính: tăng tỉ lệ giun từ Khá hiếm trở lên
	AbilityHaggle = "haggle" // Mặc Cả: giảm giá vé
	AbilityCarry  = "carry"  // Lưng Chắc: thêm giây chạy; Lv3 cưỡi ngỗng
	AbilityFeet   = "feet"   // Chân Nhanh: đi bộ nhanh hơn
	AbilityBelly  = "belly"  // Bụng Khoẻ: độ no tụt chậm hơn
	AbilityTrader = "trader" // Tay Buôn: bán giun giá cao hơn
	AbilityDouble = "double" // May Mắn Kép: đào được 2 con cùng loài
)

type SpeciesDef struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Emo         string     `json:"emo"`
	Ability     string     `json:"ability"`
	AbilityName string     `json:"abilityName"`
	Desc        string     `json:"desc"`
	Values      [3]float64 `json:"values"`
	Unit        string     `json:"unit"` // cách hiển thị giá trị: "%" hoặc "s"
	Body        string     `json:"body"` // màu thân mặc định cho mô hình
}

var Species = []SpeciesDef{
	{"ga_con", "Gà con", "🐤", AbilitySniff, "Mỏ Thính", "Đánh hơi giun: tăng tỉ lệ giun từ Khá hiếm trở lên.", [3]float64{.04, .07, .10}, "%", "#f4cf3f"},
	{"vit_con", "Vịt con", "🦆", AbilityHaggle, "Mặc Cả", "Giảm giá vé xe buýt và xe liên trại (xe ôm tối thiểu vẫn 1 thóc).", [3]float64{.20, .35, .50}, "%", "#f5f2e9"},
	{"ngong_con", "Ngỗng con", "🪿", AbilityCarry, "Lưng Chắc", "Thêm giây chạy nhanh; thân thiết cao nhất thì cõng chủ chạy ×1,5 trong 20 giây.", [3]float64{2, 3, 4}, "s", "#eef0f2"},
	{"chim_cut", "Chim cút", "🐦", AbilityFeet, "Chân Nhanh", "Đi bộ nhanh hơn.", [3]float64{.05, .08, .10}, "%", "#a07850"},
	{"ga_tre", "Gà tre", "🐓", AbilityBelly, "Bụng Khoẻ", "Độ no tụt chậm hơn.", [3]float64{.10, .18, .25}, "%", "#c8642c"},
	{"ga_ac", "Gà ác", "🐔", AbilityTrader, "Tay Buôn", "Bán giun cho Lợn Ủn giá cao hơn.", [3]float64{.05, .08, .12}, "%", "#3b3530"},
	{"cong_con", "Công con", "🦚", AbilityDouble, "May Mắn Kép", "Có cơ hội đào được 2 con cùng loài.", [3]float64{.03, .05, .08}, "%", "#3f8fa8"},
}

func speciesByID(id string) *SpeciesDef {
	for i := range Species {
		if Species[i].ID == id {
			return &Species[i]
		}
	}
	return nil
}

// petBonus là giá trị kỹ năng của bạn đồng hành đang đi theo, 0 nếu không có kỹ năng đó.
func petBonus(a *Account, ability string) float64 {
	c := a.companion(a.Following)
	if c == nil {
		return 0
	}
	sp := speciesByID(c.Species)
	if sp == nil || sp.Ability != ability {
		return 0
	}
	lv, _ := AffinityLevel(c.Affinity)
	if lv < 0 {
		lv = 0
	}
	if lv > 2 {
		lv = 2
	}
	return sp.Values[lv]
}

// Cưỡi ngỗng: Ngỗng con thân thiết cao nhất cõng chủ chạy ×1,5 trong 20 giây, hồi 3 phút.
const (
	GooseMult     = 1.5
	GooseRide     = 20 * time.Second
	GooseCooldown = 3 * time.Minute
)

// canRideGoose: đang dắt Ngỗng con ở mức thân thiết cao nhất.
func canRideGoose(a *Account) bool {
	c := a.companion(a.Following)
	if c == nil || c.Species != "ngong_con" {
		return false
	}
	lv, _ := AffinityLevel(c.Affinity)
	return lv >= 2
}

// Giới hạn cộng dồn (PLAN_LOGIC_GAME 15.2).
const (
	LuckCap     = 0.10
	DiscountCap = 0.50
)

func digLuck(a *Account) float64 { return min(LuckCap, petBonus(a, AbilitySniff)) }

// fareAfter áp giảm giá vé của "Mặc Cả", làm tròn xuống, không quá 50%.
func fareAfter(a *Account, fare int) int {
	d := min(DiscountCap, petBonus(a, AbilityHaggle))
	return int(float64(fare) * (1 - d))
}
