package main

import "time"

// Cấu hình nội dung và cân bằng. Số liệu bám PLAN.md (mục 7.2, 7.4, 8.2, 8.3).
// Client chỉ nhận bản sao để hiển thị; mọi quyết định dùng bảng ở đây.

type Worm struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Rarity string  `json:"rarity"`
	Prob   float64 `json:"prob"`
	Sell   int     `json:"sell"`
	XP     int     `json:"xp"`
	Mana   int     `json:"mana"`
	Fee    int     `json:"fee"`
	Color  string  `json:"color"`
	Rare   bool    `json:"rare"` // từ "Hiếm" trở lên: có hoạt ảnh nâng giun và nút khoe
	Region string  `json:"region,omitempty"`
}

var Worms = []Worm{
	// Ba bậc trên được tăng nhẹ so với PLAN.md 7.2 để bù giun thoát khi giằng co (PLAN_LOGIC_GAME 4.4).
	{"dat", "Giun đất", "Thường", 0.7422, 5, 10, 10, 2, "#d98a8a", false, RegionHypex},
	{"beo", "Giun béo", "Khá hiếm", 0.20, 12, 25, 20, 3, "#c86f9a", false, RegionHypex},
	{"sang", "Giun phát sáng", "Hiếm", 0.046, 40, 80, 30, 5, "#4fc9b4", true, RegionHypex},
	{"vang", "Giun vàng", "Sử thi", 0.0105, 150, 200, 40, 8, "#e0b52a", true, RegionHypex},
	{"cauvong", "Giun cầu vồng", "Huyền thoại", 0.0013, 500, 500, 60, 12, "#b65cf0", true, RegionHypex},
}

type Food struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Emo  string `json:"emo"`
	Cost int    `json:"cost"`
	Mana int    `json:"mana"`
}

// Thức ăn mua: hồi mana, 0 XP, không bán lại (mục 7.2, 8.2).
var Foods = []Food{
	{"cam", "Khẩu phần cám", "🌾", 5, 20},
	{"rau", "Đĩa rau trộn", "🥗", 12, 50},
	{"suat", "Suất ăn đầy đủ", "🍚", 20, 100},
	{"banh_thoc", "Bánh thóc mùa thu", "🥮", 15, 60}, // món ăn mới V1.2
}

type Cosmetic struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Emo    string `json:"emo"`
	Slot   string `json:"slot"` // head, eye, neck, body, back
	Cost   int    `json:"cost"`
	Kind   string `json:"kind"`
	Color  string `json:"color,omitempty"`
	Theme  string `json:"theme"`
	Season string `json:"season,omitempty"` // đồ mùa: chỉ bán trong mùa (mục 8.3, 15)
}

// Danh mục mục 8.3: váy, vest, áo, mũ, kính, nơ/khăn cổ, phụ kiện lưng.
// Mùa và sự kiện (mục 15 V1.2: bộ sưu tập mùa, giun sự kiện). Khung ngày theo giờ Việt Nam.
type Season struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Worm     string    `json:"worm,omitempty"`
	WormProb float64   `json:"wormProb,omitempty"`
}

var Seasons = []Season{
	{ID: "thu2026", Name: "Mùa thu 2026", From: time.Date(2026, 9, 1, 0, 0, 0, 0, vnZone), To: time.Date(2026, 12, 1, 0, 0, 0, 0, vnZone), Worm: "bi_ngo", WormProb: 0.005},
	{ID: "tet2027", Name: "Tết 2027", From: time.Date(2027, 1, 20, 0, 0, 0, 0, vnZone), To: time.Date(2027, 3, 1, 0, 0, 0, 0, vnZone)},
}

func CurrentSeason(now time.Time) *Season {
	for i := range Seasons {
		if !now.Before(Seasons[i].From) && now.Before(Seasons[i].To) {
			return &Seasons[i]
		}
	}
	return nil
}

// Giun sự kiện chỉ xuất hiện trong mùa, lấy tỉ lệ đều từ bảng thường nên tổng vẫn bằng 1.
var EventWorms = []Worm{
	{"bi_ngo", "Giun bí ngô", "Sử thi", 0.005, 120, 180, 40, 8, "#e9833a", true, "event"},
}

var Cosmetics = []Cosmetic{
	{"mu_rom", "Mũ rơm", "👒", "head", 220, "straw", "#d9b65a", "Nông dân", ""},
	{"mu_luoi", "Mũ lưỡi trai", "🧢", "head", 260, "cap", "#d64b3a", "Vui nhộn", ""},
	{"mu_phot", "Mũ phớt", "🎩", "head", 380, "tophat", "#2b2b33", "Lịch lãm", ""},
	{"vuong_mien", "Vương miện", "👑", "head", 400, "crown", "#e7c23a", "Dạ hội", ""},
	{"no_hoa", "Cài hoa", "🌸", "head", 200, "flower", "#f08cb4", "Dạ hội", ""},
	{"kinh_ram", "Kính râm", "🕶️", "eye", 200, "shades", "#1a1a1a", "Vui nhộn", ""},
	{"kinh_tron", "Kính tròn", "👓", "eye", 240, "round", "#3a3a3a", "Lịch lãm", ""},
	{"kinh_tim", "Kính trái tim", "💗", "eye", 300, "heart", "#e2457a", "Vui nhộn", ""},
	{"no_do", "Nơ cổ đỏ", "🎀", "neck", 200, "bow", "#d64b3a", "Lịch lãm", ""},
	{"khan_ran", "Khăn rằn", "🧣", "neck", 220, "scarf", "#3d5a80", "Nông dân", ""},
	{"ao_hoa", "Áo hoa nông dân", "🌻", "body", 520, "shirt", "#e0a53a", "Nông dân", ""},
	{"ao_thun", "Áo thun xanh", "👕", "body", 500, "shirt", "#3f86c9", "Vui nhộn", ""},
	{"vest", "Vest lịch lãm", "🤵", "body", 900, "vest", "#2d3a52", "Lịch lãm", ""},
	{"vest_trang", "Vest trắng", "🥼", "body", 1000, "vest", "#ece6d8", "Dạ hội", ""},
	{"vay_hong", "Váy dạ hội hồng", "👗", "body", 800, "dress", "#d46ea0", "Dạ hội", ""},
	{"vay_vang", "Váy hoa vàng", "💃", "body", 1200, "dress", "#e8b93a", "Dạ hội", ""},
	{"balo", "Balo nhỏ", "🎒", "back", 300, "pack", "#7b4f2d", "Nông dân", ""},
	{"canh_thien_than", "Cánh thiên thần", "🪽", "back", 1100, "wings", "#ffffff", "Dạ hội", ""},
	{"mu_len", "Mũ len đỏ", "🧶", "head", 240, "beanie", "#c0392b", "Vui nhộn", ""},
	{"no_xanh", "Nơ cổ xanh", "🎗️", "neck", 200, "bow", "#3f86c9", "Lịch lãm", ""},
	{"khan_hoa", "Khăn hoa vàng", "🌼", "neck", 260, "scarf", "#e8b93a", "Dạ hội", ""},
	{"yem_nong_dan", "Yếm nông dân", "👖", "body", 560, "overall", "#3d5a80", "Nông dân", ""},
	{"vest_do", "Vest đỏ rượu", "🧥", "body", 950, "vest", "#7a1f2b", "Lịch lãm", ""},
	{"canh_buom", "Cánh bướm", "🦋", "back", 700, "wings", "#8fd3ff", "Vui nhộn", ""},
	// Bộ sưu tập mùa thu 2026
	{"mu_bi_ngo", "Mũ bí ngô", "🎃", "head", 450, "beanie", "#e9833a", "Mùa thu", "thu2026"},
	{"khan_la_phong", "Khăn lá phong", "🍁", "neck", 380, "scarf", "#c0392b", "Mùa thu", "thu2026"},
	{"ao_len_thu", "Áo len mùa thu", "🧶", "body", 700, "shirt", "#a0522d", "Mùa thu", "thu2026"},
}

var Slots = []string{"head", "eye", "neck", "body", "back"}

// Ngoại hình khởi tạo (mục 5).
var FeatherColors = []string{"vang", "kem", "cam", "nau", "trang", "xam"}
var CombKinds = []string{"don", "kep", "ro"}
var StartShirts = []string{"none", "xanh", "do", "luc", "tim"}
var ProfileSexes = []string{"nam", "nu", "khac", "an"}
var TailKinds = []string{"quat", "nhon", "xoan"}
var EyeKinds = []string{"tron", "to", "cuoi", "mi", "sao", "ngai"}

// Cân bằng có hai bộ: "plan" = số trong PLAN.md, "demo" = rút ngắn để playtest nhanh.
type Balance struct {
	Name             string        `json:"name"`
	DigCooldown      time.Duration `json:"-"` // 1 phần thưởng đào mỗi khoảng này, theo tài khoản (mục 7.1)
	DigCooldownSec   float64       `json:"digCooldownSec"`
	DigDailyLimit    int           `json:"digDailyLimit"`
	FishDailyLimit   int           `json:"fishDailyLimit"` // số cá câu được mỗi ngày; 0 → FishDailyDefault
	DigMinDuration   time.Duration `json:"-"` // thao tác nhanh hơn mức này coi là bất thường
	ManaDrainPerSec  float64       `json:"manaDrainPerSec"`
	ManaPerDig       float64       `json:"manaPerDig"` // trừ thêm mỗi lượt đào có thưởng (mục 7.4)
	HungerSlowSec    float64       `json:"hungerSlowSec"`
	HungerExhaustSec float64       `json:"hungerExhaustSec"`
	RescueDuration   time.Duration `json:"-"`
	RescueSec        float64       `json:"rescueSec"`
	RescueReward     int           `json:"rescueReward"`
	TutorialReward   int           `json:"tutorialReward"`
	ShowcaseEvery    time.Duration `json:"-"`
	BuildDuration    time.Duration `json:"-"` // Bò Mộc xây chuồng theo mẫu (mục 9.1)
	BuildSec         float64       `json:"buildSec"`
	ReconnectGrace   time.Duration `json:"-"` // giữ chỗ khi mất kết nối (mục 6.3, 10.2)
	HatchDuration    time.Duration `json:"-"` // trứng đầu 2 giờ (mục 12)
	HatchSec         float64       `json:"hatchSec"`
	BusWait          time.Duration `json:"-"` // chờ khách tối đa trước khi xe chạy (mục 8.5)
	BusWaitSec       float64       `json:"busWaitSec"`
	ContractWork     int           `json:"contractWork"` // số lần thi công để hoàn thành mẫu chuồng
	ContractTTL      time.Duration `json:"-"`
	ContractSec      float64       `json:"contractSec"`
	OwnerAbsent      time.Duration `json:"-"` // chủ hội vắng lâu thì phó hội được nhận quyền (mục 13)
	// PLAN_LOGIC_GAME: nghỉ ổ, ăn sống, quà khởi nghiệp, xe liên trại theo lịch, nhịp giằng co.
	ManaRestFactor float64       `json:"manaRestFactor"` // ngồi ổ: độ no tụt chậm theo hệ số này
	RawEatRatio    float64       `json:"rawEatRatio"`    // ăn sống giun: phần độ no so với chế biến
	StarterThoc    int           `json:"starterThoc"`
	BusEvery       time.Duration `json:"-"` // xe liên trại xuất bến theo lịch
	BusEverySec    float64       `json:"busEverySec"`
	TugSpeed       float64       `json:"tugSpeed"` // >1: giằng co nhanh hơn (chỉ dùng cho test)
	DigPeckMs      int           `json:"digPeckMs"`
	DigRevealMs    int           `json:"digRevealMs"`
	TokenEvery     time.Duration `json:"-"` // hồi 1 lượt xe miễn phí
	TokenEverySec  float64       `json:"tokenEverySec"`
	BusLoopSpeed   float64       `json:"busLoopSpeed"` // xe buýt vòng làng (m/giây)
	BusLoopDwell   time.Duration `json:"-"`            // dừng ở mỗi bến
	BusDwellSec    float64       `json:"busDwellSec"`
	JoinGiftsOff   bool          `json:"-"` // chỉ test: tắt điểm danh và quà cấp còn nợ lúc vào game
}

func PlanBalance() Balance {
	return finalize(Balance{
		Name: "plan", DigCooldown: 45 * time.Second, DigDailyLimit: 120, DigMinDuration: 400 * time.Millisecond,
		ManaDrainPerSec: 1.0 / 60, ManaPerDig: 1, HungerSlowSec: 300, HungerExhaustSec: 600,
		RescueDuration: 60 * time.Second, RescueReward: 5, TutorialReward: 100, ShowcaseEvery: 60 * time.Second,
		BuildDuration: 10 * time.Minute, HatchDuration: 2 * time.Hour, BusWait: 30 * time.Second,
		ContractWork: 10, ContractTTL: 24 * time.Hour, OwnerAbsent: 30 * 24 * time.Hour,
	})
}

// LiveBalance là bộ mặc định khi chơi thật (PLAN_LOGIC_GAME mục 17): độ no 15–20 phút, đào ~5 lượt/phút.
func LiveBalance() Balance {
	return finalize(Balance{
		Name: "live", DigCooldown: 8 * time.Second, DigDailyLimit: 120, DigMinDuration: 400 * time.Millisecond,
		ManaDrainPerSec: 4.5 / 60, ManaPerDig: 0.4, HungerSlowSec: 180, HungerExhaustSec: 360,
		RescueDuration: 30 * time.Second, RescueReward: 5, TutorialReward: 50, ShowcaseEvery: 60 * time.Second,
		BuildDuration: 10 * time.Minute, HatchDuration: 30 * time.Minute, BusWait: 20 * time.Second, BusEvery: 3 * time.Minute,
		ContractWork: 6, ContractTTL: 24 * time.Hour, OwnerAbsent: 7 * 24 * time.Hour,
	})
}

// DemoBalance rút ngắn các mốc chờ để thử nhanh; độ no dùng chung số với bộ live (người chơi thử
// phàn nàn 0,5 điểm/giây hết quá nhanh).
func DemoBalance() Balance {
	return finalize(Balance{
		Name: "demo", DigCooldown: 4 * time.Second, DigDailyLimit: 120, DigMinDuration: 400 * time.Millisecond,
		ManaDrainPerSec: 4.5 / 60, ManaPerDig: 0.4, HungerSlowSec: 180, HungerExhaustSec: 360, BusEvery: 90 * time.Second,
		RescueDuration: 8 * time.Second, RescueReward: 5, TutorialReward: 50, ShowcaseEvery: 60 * time.Second,
		BuildDuration: 20 * time.Second, HatchDuration: 30 * time.Second, BusWait: 10 * time.Second,
		ContractWork: 4, ContractTTL: 10 * time.Minute, OwnerAbsent: 10 * time.Minute,
	})
}

// FishDailyDefault: trần cá mỗi ngày. Kỳ vọng ~12 thóc/con → ~720 thóc/ngày, dưới ngân sách ~1.150 thóc
// của 120 lượt đào (PLAN_LOGIC_GAME 17). Không có trần thì câu cá thành nguồn thóc/XP vô hạn (~2.800 thóc/giờ).
const FishDailyDefault = 60

func finalize(b Balance) Balance {
	b.DigCooldownSec = b.DigCooldown.Seconds()
	if b.FishDailyLimit == 0 {
		b.FishDailyLimit = FishDailyDefault
	}
	b.RescueSec = b.RescueDuration.Seconds()
	b.BuildSec = b.BuildDuration.Seconds()
	if b.ReconnectGrace == 0 {
		b.ReconnectGrace = 15 * time.Second
	}
	if b.HatchDuration == 0 {
		b.HatchDuration = time.Second
	}
	if b.BusWait == 0 {
		b.BusWait = time.Second
	}
	if b.ContractWork == 0 {
		b.ContractWork = 2
	}
	if b.ContractTTL == 0 {
		b.ContractTTL = time.Minute
	}
	if b.OwnerAbsent == 0 {
		b.OwnerAbsent = time.Hour
	}
	if b.ManaRestFactor == 0 {
		b.ManaRestFactor = 0.5
	}
	if b.RawEatRatio == 0 {
		b.RawEatRatio = 0.6
	}
	if b.StarterThoc == 0 {
		b.StarterThoc = 100
	}
	if b.BusEvery == 0 {
		b.BusEvery = 3 * time.Minute
	}
	if b.TugSpeed == 0 {
		b.TugSpeed = 1
	}
	if b.DigPeckMs == 0 {
		b.DigPeckMs = 900
	}
	if b.DigRevealMs == 0 {
		b.DigRevealMs = 800
	}
	if b.TokenEvery == 0 {
		b.TokenEvery = 5 * time.Minute
	}
	if b.BusLoopSpeed == 0 {
		b.BusLoopSpeed = 9
	}
	if b.BusLoopDwell == 0 {
		b.BusLoopDwell = 8 * time.Second
	}
	b.BusDwellSec = b.BusLoopDwell.Seconds()
	b.HatchSec, b.BusWaitSec, b.ContractSec = b.HatchDuration.Seconds(), b.BusWait.Seconds(), b.ContractTTL.Seconds()
	b.BusEverySec, b.TokenEverySec = b.BusEvery.Seconds(), b.TokenEvery.Seconds()
	return b
}

const (
	LevelCap      = 20 // PLAN_LOGIC_GAME 16: mở cấp 11–20
	ManaMax       = 100.0
	ManaHungryAt  = 20.0 // dưới mức này là "sắp đói"; ăn lên >= mức này gỡ phạt
	WalkSpeed     = 4.6  // bản đồ liền rộng hơn: nhanh hơn 10% so với 4,2 ban đầu
	RoomCapacity  = 50   // public channel; zone capacities are independently enforced
	TickRate      = 20
	SnapEvery     = 2 // gửi snapshot mỗi 2 tick = 10 lần/giây
	ChatMaxLen    = 120
	NameMinLen    = 2
	NameMaxLen    = 14
	InteractSlack = 1.2 // dung sai khoảng cách tương tác cho độ trễ mạng
	BagCapacity   = 60  // tổng số giun trong túi
	PKInviteTTL   = 20 * time.Second
	PeckDuration  = 8 * time.Second
	PKCountdown   = 3 * time.Second
	PeckMinGap    = 60 * time.Millisecond // giới hạn ~16 lần mổ hợp lệ/giây
	CoinRounds    = 3
)

// Chạy nhanh (PLAN_LOGIC_GAME mục 3): 6 giây thể lực, hụt hơi thì khoá 5 giây rồi đầy lại.
const (
	SprintMult      = 1.65
	StaminaSec      = 6.0
	StaminaLock     = 5 * time.Second
	StaminaRegenGap = 800 * time.Millisecond
	StaminaRegen    = 0.25 // phần thể lực tối đa hồi mỗi giây khi thả nút sớm
)

// XP để đi từ cấp L lên L+1 (mục 7.3).
// LevelXP: cấp 1–9 giữ công thức PLAN.md (80 + 40 × (L−1)); từ cấp 10 dốc hơn (440 + 90 × (L−10)) để cấp
// 11–20 là chặng đường dài cho người chơi lâu năm.
func LevelXP(level int) int {
	if level < 10 {
		return 80 + 40*(level-1)
	}
	return 440 + 90*(level-10)
}

// Chủ đề tên chuồng công cộng (mục 6.4).
var RoomThemes = []string{"Chuồng Hạt Thóc", "Chuồng Rơm Vàng", "Chuồng Bình Minh"}

// Giun đặc sản trại Ven Suối (mục 7.2.1, V1.1): bảng xác suất riêng, có hiển thị.
var VenSuoiWorms = []Worm{
	{"suoi", "Giun suối", "Thường", 0.6915, 8, 14, 12, 2, "#c99b8a", false, RegionVenSuoi},
	{"bun", "Giun bùn", "Khá hiếm", 0.25, 18, 30, 22, 3, "#7d6450", false, RegionVenSuoi},
	{"ngoc", "Giun ngọc", "Hiếm", 0.052, 60, 110, 35, 6, "#5ad1e0", true, RegionVenSuoi},
	{"rong", "Giun rồng", "Huyền thoại", 0.0065, 400, 450, 60, 12, "#e2452e", true, RegionVenSuoi},
}

var WormSets = map[string][]Worm{RegionHypex: Worms, RegionVenSuoi: VenSuoiWorms}

var AllWorms = append(append(append([]Worm{}, Worms...), VenSuoiWorms...), EventWorms...)

func wormByID(id string) *Worm {
	for i := range AllWorms {
		if AllWorms[i].ID == id {
			return &AllWorms[i]
		}
	}
	return nil
}

// Phương tiện cá nhân (mục 8.4): tổng ghế gồm người lái; giá ở bảng 8.2.
type VehicleDef struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Emo   string  `json:"emo"`
	Cost  int     `json:"cost"`
	Seats int     `json:"seats"`
	Level int     `json:"level"`
	Speed float64 `json:"speed"` // bội số tốc độ chạy bộ
	Road  bool    `json:"road"`  // đi được đường liên trại
	Phase string  `json:"phase"`
}

var Vehicles = []VehicleDef{
	{"xe_dap", "Xe đạp có ghế sau", "🚲", 800, 2, 1, 1.3, false, "V1.1"},
	{"xe_may", "Xe máy", "🛵", 3000, 2, 5, 1.7, true, "V1.2"},
	{"o_to", "Ô tô nhỏ", "🚗", 10000, 4, 7, 2.0, true, "V1.2"},
}

func vehicleByID(id string) *VehicleDef {
	for i := range Vehicles {
		if Vehicles[i].ID == id {
			return &Vehicles[i]
		}
	}
	return nil
}

const (
	BusSeats      = 8  // hành khách mỗi chuyến, tài xế NPC riêng
	FreeTrips     = 3  // lượt miễn phí/ngày/tài khoản, reset 0:00 Asia/Ho_Chi_Minh
	BusFare       = 10 // tuyến gần Hypex ↔ Ven Suối
	CompanionMax  = 3
	GuildMax      = 30
	GuildLevel    = 10
	GuildCost     = 1000
	TradeLevel    = 3 // chặn chuyển phần thưởng từ tài khoản mới (mục 8.2)
	ContractLevel = 7 // mở nghề và hợp đồng xây dựng (mục 7.3)
)

// Mốc nhận trứng qua nhiệm vụ của Vịt Bông (mục 12: chưa bán trứng ngẫu nhiên).
var EggMilestones = []int{5, 7, 9}

func foodByID(id string) *Food {
	for i := range Foods {
		if Foods[i].ID == id {
			return &Foods[i]
		}
	}
	return nil
}

func cosmeticByID(id string) *Cosmetic {
	for i := range Cosmetics {
		if Cosmetics[i].ID == id {
			return &Cosmetics[i]
		}
	}
	return nil
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
