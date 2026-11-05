package main

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Account là tài sản và tiến độ lâu dài của người chơi. Chỉ goroutine của phòng
// đang giữ phiên được sửa nó; mọi thay đổi tài sản được lưu trước khi báo thành công.

type Character struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Comb  string `json:"comb"`
	Shirt string `json:"shirt"`
	Sex   string `json:"sex"`
	Tail  string `json:"tail"`
	Eyes  string `json:"eyes"`
}

type Quest struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Need   int    `json:"need"`
	Cur    int    `json:"cur"`
	Reward int    `json:"reward"`
	Done   bool   `json:"done"`
}

// Bản ghi một lần đào trên máy chủ: nguồn duy nhất cho thẻ khoe (mục 7.2.2).
type Loot struct {
	ID      string    `json:"id"`
	Worm    string    `json:"worm"`
	SizeCM  float64   `json:"size"`
	Zone    string    `json:"zone"`
	Spot    string    `json:"spot"`
	At      time.Time `json:"at"`
	Shared  bool      `json:"shared"`
	Perfect bool      `json:"perfect"`
}

type RescueJob struct {
	ID      string    `json:"id"`
	StartAt time.Time `json:"startAt"`
}

type Stats struct {
	Dug      int `json:"dug"`
	Escaped  int `json:"escaped,omitempty"` // giun thoát khi giằng co
	RawEaten int `json:"rawEaten,omitempty"`
	Perfect  int `json:"perfect"`
	Sold     int `json:"sold"`
	Eaten    int `json:"eaten"`
	Shows    int `json:"shows"`
	PKPlayed int `json:"pkPlayed"`
	PKWins   int `json:"pkWins"`
}

type ActResult struct {
	OK   bool           `json:"ok"`
	Err  string         `json:"err,omitempty"`
	Msg  string         `json:"msg,omitempty"`
	Data map[string]any `json:"data,omitempty"`
}

type LedgerEntry struct {
	At      time.Time `json:"at"`
	Account string    `json:"account"`
	Delta   int       `json:"delta"`
	Balance int       `json:"balance"`
	Reason  string    `json:"reason"`
	Ref     string    `json:"ref"`
}

type Account struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	CreatedAt time.Time  `json:"createdAt"`
	Char      *Character `json:"char"`

	Level       int     `json:"level"`
	XP          int     `json:"xp"`
	Thoc        int     `json:"thoc"`
	Mana        float64 `json:"mana"`
	ZeroElapsed float64 `json:"zeroElapsed"` // giây online tại mana=0 kể từ lần hồi đạt 20 gần nhất

	Worms      map[string]int      `json:"worms"`
	Food       map[string]int      `json:"food"`
	Collection map[string]bool     `json:"collection"`
	Records    map[string]float64  `json:"records"` // kỷ lục chiều dài theo loài
	Owned      map[string]bool     `json:"owned"`
	Equipped   map[string]string   `json:"equipped"`
	Outfits    []map[string]string `json:"outfits"`

	DigReadyAt time.Time `json:"digReadyAt"`
	DailyKey   string    `json:"dailyKey"`
	DailyDigs  int       `json:"dailyDigs"`
	DailyFish  int       `json:"dailyFish,omitempty"` // cá câu được hôm nay (trần Balance.FishDailyLimit)
	// Quà chỉ đường cho shipper (+1 thóc, tối đa ShipperTipsPerDay/ngày). Lưu theo tài khoản, không theo phiên:
	// trước đây đổi kênh là đếm lại từ 0 nên farm được thóc vô hạn.
	ShipTipDay string `json:"shipTipDay,omitempty"`
	ShipTips   int    `json:"shipTips,omitempty"`
	Daily      []Quest   `json:"daily"`

	Tutorial        map[string]bool `json:"tutorial"`
	TutorialClaimed bool            `json:"tutorialClaimed"`
	TutorialSkipped bool            `json:"tutorialSkipped,omitempty"` // bỏ qua hướng dẫn: giữ quà khởi nghiệp, không nhận thưởng hướng dẫn
	Rescue          *RescueJob      `json:"rescue"`
	RescueDone      map[string]bool `json:"rescueDone"`
	LastShowAt      time.Time       `json:"lastShowAt"`

	Stats Stats  `json:"stats"`
	Loots []Loot `json:"loots"`

	Discover   map[string]bool `json:"discover"`   // sổ khám phá: khu đã tới, NPC đã gặp (mục 11)
	Titles     []string        `json:"titles"`     // danh hiệu đã mở
	Title      string          `json:"title"`      // danh hiệu đang đeo
	OrdersDone map[string]bool `json:"ordersDone"` // "ngày:mã đơn" của bảng việc làng
	Festival   int             `json:"festival"`   // thóc đã góp cho ngày hội

	// V1.1
	Eggs          int             `json:"eggs"`
	EggsClaimed   map[int]bool    `json:"eggsClaimed"`
	Incubation    *Incubation     `json:"incubation"`
	Companions    []Companion     `json:"companions"`
	Following     string          `json:"following"`
	Booking       *Booking        `json:"booking"`
	TravelDay     string          `json:"travelDay"`
	TripsUsed     int             `json:"tripsUsed"`
	OwnedVehicles map[string]bool `json:"ownedVehicles"`
	Reputation    int             `json:"reputation"`
	RepPairs      map[string]bool `json:"repPairs"`
	// PLAN_LOGIC_GAME: quà khởi nghiệp một lần, bữa cứu trợ mỗi ngày của Cụ Cục.
	StarterGranted bool                 `json:"starterGranted,omitempty"`
	RationDay      string               `json:"rationDay,omitempty"`
	Items          map[string]int       `json:"items"`         // kho: vật liệu, hạt giống, nông sản, trứng
	LevelRewarded  map[int]bool         `json:"levelRewarded"` // cấp đã nhận quà
	Coupons        map[string]time.Time `json:"coupons"`       // mã phiếu → hạn dùng
	BusTokens      int                  `json:"busTokens"`     // lượt xe miễn phí đang có (tối đa 2)
	BusTokenAt     time.Time            `json:"busTokenAt"`    // mốc bắt đầu tính lượt hồi kế tiếp
	TokensInit     bool                 `json:"tokensInit,omitempty"`
	Prefs          Prefs                `json:"prefs"`
	NPCSeen        map[string]string    `json:"npcSeen"` // khoá lời chào NPC → ngày đã chào đầy đủ
	CheckIn        CheckIn              `json:"checkIn"`
	Beds           map[string]*Plant    `json:"beds"`                 // luống đang trồng: mã luống → cây
	HouseBuilt     bool                 `json:"houseBuilt,omitempty"` // nhà riêng đã xây xong (mở chuồng nuôi thú cưng)
	Week           WeekStats            `json:"week"`                 // thành tích tuần cho bảng xếp hạng
	FishReadyAt    time.Time            `json:"fishReadyAt"`          // câu cá: hồi câu
	FishCaught     int                  `json:"fishCaught,omitempty"`

	Recent      map[string]ActResult `json:"recent"` // request ID → kết quả (idempotency)
	RecentOrder []string             `json:"recentOrder"`

	ledger []LedgerEntry
	seq    int
}

func NewAccount(id, username string, now time.Time) *Account {
	a := &Account{ID: id, Username: username, CreatedAt: now, Level: 1, Mana: ManaMax}
	a.normalize()
	return a
}

// normalize bảo đảm map/slice không nil sau khi đọc bản lưu cũ.
func (a *Account) normalize() {
	if a.Worms == nil {
		a.Worms = map[string]int{}
	}
	if a.Food == nil {
		a.Food = map[string]int{}
	}
	if a.Collection == nil {
		a.Collection = map[string]bool{}
	}
	if a.Records == nil {
		a.Records = map[string]float64{}
	}
	if a.Owned == nil {
		a.Owned = map[string]bool{}
	}
	if a.Equipped == nil {
		a.Equipped = map[string]string{}
	}
	for len(a.Outfits) < 3 {
		a.Outfits = append(a.Outfits, nil)
	}
	if a.Tutorial == nil {
		a.Tutorial = map[string]bool{}
	}
	if a.RescueDone == nil {
		a.RescueDone = map[string]bool{}
	}
	if a.Recent == nil {
		a.Recent = map[string]ActResult{}
	}
	if a.Discover == nil {
		a.Discover = map[string]bool{}
	}
	if a.OrdersDone == nil {
		a.OrdersDone = map[string]bool{}
	}
	if a.EggsClaimed == nil {
		a.EggsClaimed = map[int]bool{}
	}
	if a.OwnedVehicles == nil {
		a.OwnedVehicles = map[string]bool{}
	}
	if a.RepPairs == nil {
		a.RepPairs = map[string]bool{}
	}
	if a.Items == nil {
		a.Items = map[string]int{}
	}
	if a.LevelRewarded == nil {
		a.LevelRewarded = map[int]bool{}
	}
	if a.Coupons == nil {
		a.Coupons = map[string]time.Time{}
	}
	if a.NPCSeen == nil {
		a.NPCSeen = map[string]string{}
	}
	if a.Beds == nil {
		a.Beds = map[string]*Plant{}
	}
	if a.Eggs > 0 {
		// Bản lưu cũ: trứng là một con số; chuyển vào kho thành trứng thường.
		a.Items["trung_thuong"] += a.Eggs
		a.Eggs = 0
	}
	if !a.TokensInit {
		a.TokensInit, a.BusTokens = true, MaxBusTokens
	}
	if a.Level < 1 {
		a.Level = 1
	}
}

func (a *Account) nextID(prefix string) string {
	a.seq++
	return fmt.Sprintf("%s_%s_%d_%d", prefix, a.ID[:6], time.Now().UnixNano(), a.seq)
}

// ---------- Ví ----------

var ErrNotEnough = errors.New("Không đủ thóc.")

func (a *Account) addThoc(delta int, reason, ref string, now time.Time) error {
	if a.Thoc+delta < 0 {
		return ErrNotEnough
	}
	a.Thoc += delta
	a.ledger = append(a.ledger, LedgerEntry{At: now, Account: a.ID, Delta: delta, Balance: a.Thoc, Reason: reason, Ref: ref})
	return nil
}

func (a *Account) TakeLedger() []LedgerEntry {
	l := a.ledger
	a.ledger = nil
	return l
}

// ---------- XP / cấp ----------

func (a *Account) addXP(n int) (levelsGained int) {
	if a.Level >= LevelCap {
		a.XP = 0
		return 0
	}
	a.XP += n
	for a.Level < LevelCap && a.XP >= LevelXP(a.Level) {
		a.XP -= LevelXP(a.Level)
		a.Level++
		levelsGained++
	}
	if a.Level >= LevelCap {
		a.XP = 0
	}
	return
}

// ---------- Mana / đói (mục 7.4) ----------

// Stage: 0 bình thường, 1 sắp đói, 2 đói, 3 đói kéo dài, 4 kiệt sức.
func (a *Account) HungerStage(b Balance) int {
	if a.Mana >= ManaHungryAt {
		return 0
	}
	// Phạt đã tích lũy được giữ cho tới khi mana đạt lại ngưỡng 20.
	if a.ZeroElapsed >= b.HungerExhaustSec {
		return 4
	}
	if a.ZeroElapsed >= b.HungerSlowSec {
		return 3
	}
	if a.Mana <= 0 {
		return 2
	}
	return 1
}

// TickVitals chỉ chạy khi phiên online đang kết nối; offline không cộng dồn.
func (a *Account) TickVitals(dt float64, b Balance) { a.TickVitalsAt(dt, b, 1) }

// TickVitalsAt nhân tốc độ tụt với factor: ngồi ổ (ManaRestFactor), bạn đồng hành "Bụng Khoẻ"…
func (a *Account) TickVitalsAt(dt float64, b Balance, factor float64) {
	a.Mana = math.Max(0, a.Mana-b.ManaDrainPerSec*factor*dt)
	if a.Mana <= 0 {
		a.ZeroElapsed += dt
	}
}

func (a *Account) restoreMana(n float64) {
	a.Mana = math.Min(ManaMax, a.Mana+n)
	if a.Mana >= ManaHungryAt {
		a.ZeroElapsed = 0
	}
}

func SpeedFactor(stage int) float64 {
	switch {
	case stage >= 4:
		return 0.75 // kiệt sức: chậm 25%
	case stage >= 3:
		return 0.85 // đói kéo dài: chậm 15%
	}
	return 1
}

// ---------- Ngày & nhiệm vụ ----------

var vnZone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*3600)
}()

func DayKey(now time.Time) string { return now.In(vnZone).Format("2006-01-02") }

func (a *Account) EnsureDaily(now time.Time) {
	k := DayKey(now)
	if a.DailyKey == k {
		return
	}
	a.DailyKey = k
	a.DailyDigs = 0
	a.DailyFish = 0
	// Tổng thưởng 3 nhiệm vụ ngày: 60 thóc (mục 8.2).
	a.Daily = []Quest{
		{ID: "dig", Text: "Đào 5 giun ở Vườn giun", Need: 5, Reward: 20},
		{ID: "sell", Text: "Bán 3 con giun ở Chợ thóc", Need: 3, Reward: 20},
		{ID: "eat", Text: "Chế biến & ăn 1 con giun", Need: 1, Reward: 20},
	}
}

func (a *Account) bumpDaily(id string, now time.Time, notes *[]string) {
	a.EnsureDaily(now)
	for i := range a.Daily {
		q := &a.Daily[i]
		if q.ID != id || q.Done {
			continue
		}
		q.Cur++
		if q.Cur >= q.Need {
			q.Done = true
			_ = a.addThoc(q.Reward, "quest_daily", a.DailyKey+":"+q.ID, now)
			*notes = append(*notes, fmt.Sprintf("Hoàn thành nhiệm vụ “%s”: +%d thóc", q.Text, q.Reward))
		}
	}
}

// Các bước hướng dẫn 15 phút đầu (mục 4).
// Hướng dẫn người mới qua NPC (PLAN_LOGIC_GAME 6): mỗi bước có một NPC đội dấu "!" chỉ việc tiếp theo.
var TutorialSteps = []struct{ ID, Text, NPC, Area string }{
	{"met_elder", "Gặp Gà Cụ Cục trên bục cuối sảnh", "ga_cu_cuc", "lobby"},
	{"first_run", "Giữ Shift (hoặc nút 🏃) chạy ra cửa sảnh", "ga_don_khach", "lobby"},
	{"reach_garden", "Đi bộ hoặc đi xe buýt tới Vườn giun", "lon_un", "garden"},
	{"first_dig", "Đào con giun đầu tiên (giữ nút để kéo)", "lon_un", "garden"},
	{"first_sell", "Bán 1 con giun cho Lợn Ủn ở Chợ thóc", "sap_giun", "market"},
	{"first_meal", "Ăn một món (mua ở chợ hoặc ăn sống giun)", "quay_an", "market"},
}

// NextTutorial là bước hướng dẫn đầu tiên chưa xong (nil khi đã xong hết).
func (a *Account) NextTutorial() *struct{ ID, Text, NPC, Area string } {
	for i := range TutorialSteps {
		if !a.Tutorial[TutorialSteps[i].ID] {
			return &TutorialSteps[i]
		}
	}
	return nil
}

func (a *Account) step(id string) { a.Tutorial[id] = true }

func (a *Account) tutorialReady() bool {
	for _, s := range TutorialSteps {
		if !a.Tutorial[s.ID] {
			return false
		}
	}
	return true
}

// ---------- Idempotency ----------

func (a *Account) Remembered(rid string) (ActResult, bool) {
	if rid == "" {
		return ActResult{}, false
	}
	r, ok := a.Recent[rid]
	return r, ok
}

func (a *Account) Remember(rid string, r ActResult) {
	if rid == "" {
		return
	}
	if _, ok := a.Recent[rid]; !ok {
		a.RecentOrder = append(a.RecentOrder, rid)
	}
	a.Recent[rid] = r
	for len(a.RecentOrder) > 64 {
		delete(a.Recent, a.RecentOrder[0])
		a.RecentOrder = a.RecentOrder[1:]
	}
}

// ---------- Đào giun (mục 7.1, 7.2) ----------

func RollWorm(r float64) *Worm { return RollWormIn(RegionHypex, r) }

// RollWormIn chọn giun theo bảng của vùng; r là số ngẫu nhiên [0,1) do máy chủ tạo.
func RollWormIn(set string, r float64) *Worm {
	tab := WormSets[set]
	if tab == nil {
		tab = Worms
	}
	acc := 0.0
	for i := range tab {
		acc += tab[i].Prob
		if r < acc {
			return &tab[i]
		}
	}
	return &tab[0]
}

func (a *Account) CanDig(now time.Time, b Balance) error {
	a.EnsureDaily(now)
	// Kiệt sức vẫn đào được (chỉ ra giun thường, xem RollDigWorm) để không bao giờ kẹt (PLAN_LOGIC_GAME 2.3).
	switch {
	case a.BagCount() >= BagCapacity:
		// Túi đầy: dừng lượt trước khi bắt đầu, không âm thầm xoá vật phẩm (mục 7.1).
		return fmt.Errorf("Túi đầy (%d/%d). Bán, ăn hoặc chế biến bớt giun rồi đào tiếp.", a.BagCount(), BagCapacity)
	case now.Before(a.DigReadyAt):
		return fmt.Errorf("Đất chưa hồi, chờ %.0f giây nữa.", math.Ceil(a.DigReadyAt.Sub(now).Seconds()))
	case a.DailyDigs >= b.DigDailyLimit:
		return fmt.Errorf("Đã đạt %d lượt đào có thưởng hôm nay.", b.DigDailyLimit)
	}
	return nil
}

// FinishDig cấp phần thưởng của một lượt đào hợp lệ. rollR và sizeR là số ngẫu nhiên [0,1) do máy chủ tạo.
func (a *Account) FinishDig(zone, spot string, perfect bool, rollR, sizeR float64, now time.Time, b Balance) (Loot, []string) {
	return a.FinishDigIn(RegionHypex, zone, spot, perfect, rollR, sizeR, now, b)
}

// RollWormAt áp dụng giun sự kiện của mùa: r < WormProb ra giun sự kiện, phần còn lại co lại theo tỉ lệ.
func RollWormAt(set string, r float64, now time.Time) *Worm {
	if s := CurrentSeason(now); s != nil && s.Worm != "" {
		if r < s.WormProb {
			return wormByID(s.Worm)
		}
		r = (r - s.WormProb) / (1 - s.WormProb)
	}
	return RollWormIn(set, r)
}

func (a *Account) FinishDigIn(set, zone, spot string, perfect bool, rollR, sizeR float64, now time.Time, b Balance) (Loot, []string) {
	loot, notes, _ := a.FinishDigWorm(RollWormAt(set, rollR, now), zone, spot, perfect, sizeR, now, b, 1)
	return loot, notes
}

// FinishDigWorm cấp con giun máy chủ đã gieo lúc bắt đầu (giằng co thành công). cdMult kéo dài hồi đất
// khi đào lúc kiệt sức. Trả thêm số cấp vừa lên vì đào cũng cho XP.
func (a *Account) FinishDigWorm(w *Worm, zone, spot string, perfect bool, sizeR float64, now time.Time, b Balance, cdMult float64) (Loot, []string, int) {
	var notes []string
	size := math.Round((8+sizeR*22)*(1+0.15*float64(rarityIndex(w.ID)))*10) / 10
	loot := Loot{ID: a.nextID("loot"), Worm: w.ID, SizeCM: size, Zone: zone, Spot: spot, At: now, Perfect: perfect}
	a.Worms[w.ID]++
	if !a.Collection[w.ID] {
		notes = append(notes, "Sổ giun: khám phá loài mới — "+w.Name+"!")
	}
	a.Collection[w.ID] = true
	if size > a.Records[w.ID] {
		if a.Records[w.ID] > 0 {
			notes = append(notes, fmt.Sprintf("Kỷ lục mới: %s dài %.1f cm", w.Name, size))
		}
		a.Records[w.ID] = size
	}
	a.Stats.Dug++
	if perfect {
		a.Stats.Perfect++
	}
	a.DailyDigs++
	a.DigReadyAt = now.Add(time.Duration(float64(b.DigCooldown) * cdMult))
	a.Mana = math.Max(0, a.Mana-b.ManaPerDig)
	a.Loots = append(a.Loots, loot)
	if len(a.Loots) > 30 {
		a.Loots = a.Loots[len(a.Loots)-30:]
	}
	a.step("first_dig")
	a.bumpDaily("dig", now, &notes)
	lv := a.addXP(DigXP[wormTier(w)])
	return loot, notes, lv
}

// EscapeDig: giun thoát. Không tính vào lượt đào có thưởng của ngày, vẫn hồi đất và tốn sức; +XP an ủi.
func (a *Account) EscapeDig(now time.Time, b Balance, cdMult float64) int {
	a.DigReadyAt = now.Add(time.Duration(float64(b.DigCooldown) * cdMult))
	a.Mana = math.Max(0, a.Mana-b.ManaPerDig)
	a.Stats.Escaped++
	return a.addXP(EscapeXP)
}

func rarityIndex(id string) int {
	for _, tab := range WormSets {
		for i, w := range tab {
			if w.ID == id {
				return i
			}
		}
	}
	return 0
}

// ---------- Bán / chế biến & ăn ----------

func (a *Account) SellWorm(id string, now time.Time) (int, []string, error) {
	w := wormByID(id)
	if w == nil {
		return 0, nil, errors.New("Loại giun không hợp lệ.")
	}
	if a.Worms[id] <= 0 {
		return 0, nil, errors.New("Không còn con " + w.Name + " nào trong túi.")
	}
	var notes []string
	a.Worms[id]--
	price := SellPrice(a, w, now)
	_ = a.addThoc(price, "sell_worm", id, now)
	a.Stats.Sold++
	a.step("first_sell")
	a.bumpDaily("sell", now, &notes)
	return price, notes, nil
}

// SellPrice: giá Lợn Ủn trả cho một con giun, đã cộng loài thu mua giá cao hôm nay (+10%) và "Tay Buôn".
func SellPrice(a *Account, w *Worm, now time.Time) int {
	mult := 1.0
	if a != nil {
		mult += petBonus(a, AbilityTrader)
	}
	if w.Region != "" && HotWorm(w.Region, now) == w.ID {
		mult += 0.10
	}
	if mult == 1 {
		return w.Sell
	}
	return int(float64(w.Sell)*mult + 0.5)
}

// EatWorm là giao dịch nguyên tử: trừ đúng 1 con và phí, cộng XP/mana; thiếu điều kiện thì không đổi gì.
func (a *Account) EatWorm(id string, now time.Time) (int, []string, error) {
	w := wormByID(id)
	if w == nil {
		return 0, nil, errors.New("Loại giun không hợp lệ.")
	}
	if a.Worms[id] <= 0 {
		return 0, nil, errors.New("Không còn con " + w.Name + " nào trong túi.")
	}
	if a.Mana >= ManaMax {
		return 0, nil, errors.New("Mana đã đầy, chưa ăn được.")
	}
	if a.Thoc < w.Fee {
		return 0, nil, fmt.Errorf("Cần %d thóc phí chế biến.", w.Fee)
	}
	var notes []string
	_ = a.addThoc(-w.Fee, "process_fee", id, now)
	a.Worms[id]--
	a.restoreMana(float64(w.Mana))
	lv := a.addXP(w.XP)
	a.Stats.Eaten++
	a.step("first_meal")
	a.bumpDaily("eat", now, &notes)
	return lv, notes, nil
}

// EatWormRaw: gà ăn sống giun ngay tại chỗ, không tốn phí chế biến, không cộng XP (PLAN_LOGIC_GAME 2.3).
func (a *Account) EatWormRaw(id string, b Balance, now time.Time) (int, []string, error) {
	w := wormByID(id)
	if w == nil {
		return 0, nil, errors.New("Loại giun không hợp lệ.")
	}
	if a.Worms[id] <= 0 {
		return 0, nil, errors.New("Không còn con " + w.Name + " nào trong túi.")
	}
	if a.Mana >= ManaMax {
		return 0, nil, errors.New("Bụng đã no căng, chưa ăn được.")
	}
	var notes []string
	gain := int(math.Round(float64(w.Mana) * b.RawEatRatio))
	a.Worms[id]--
	a.restoreMana(float64(gain))
	a.Stats.RawEaten++
	a.step("first_meal") // nhiệm vụ ngày "Chế biến & ăn" chỉ tính món chế biến
	return gain, notes, nil
}

// GrantStarter cấp quà khởi nghiệp đúng một lần cho mỗi tài khoản (mới tạo hoặc có từ trước bản cập nhật).
func (a *Account) GrantStarter(now time.Time, b Balance) bool {
	if a.StarterGranted || a.Char == nil || b.StarterThoc <= 0 {
		return false
	}
	a.StarterGranted = true
	_ = a.addThoc(b.StarterThoc, "starter_gift", "starter", now)
	return true
}

// ElderRation: mỗi ngày một khẩu phần cám khi đói, gần hết thóc và không còn gì ăn hay bán.
func (a *Account) ElderRation(now time.Time) error {
	day := DayKey(now)
	switch {
	case a.RationDay == day:
		return errors.New("Hôm nay cụ đã cho cháu một phần rồi. Ra vườn đào giun đi!")
	case a.Mana >= 10 || a.Thoc >= 5 || a.hasEdibleOrSellable():
		return errors.New("Cháu vẫn còn cách tự lo được: bán giun, mua đồ ăn hoặc ăn sống một con giun nhé.")
	}
	a.RationDay = day
	a.Food["cam"]++
	return nil
}

func (a *Account) BuyFood(id string, eatNow bool, now time.Time) error {
	f := foodByID(id)
	if f == nil {
		return errors.New("Món không hợp lệ.")
	}
	if eatNow && a.Mana >= ManaMax {
		return errors.New("Mana đã đầy, chưa ăn được.")
	}
	if err := a.addThoc(-f.Cost, "buy_food", id, now); err != nil {
		return err
	}
	if eatNow {
		a.restoreMana(float64(f.Mana))
		a.step("first_meal")
	} else {
		a.Food[id]++
	}
	return nil
}

func (a *Account) EatFood(id string) error {
	f := foodByID(id)
	if f == nil || a.Food[id] <= 0 {
		return errors.New("Không có món này trong túi.")
	}
	if a.Mana >= ManaMax {
		return errors.New("Mana đã đầy, chưa ăn được.")
	}
	a.Food[id]--
	a.restoreMana(float64(f.Mana))
	a.step("first_meal")
	return nil
}

func (a *Account) hasEdibleOrSellable() bool {
	for _, n := range a.Worms {
		if n > 0 {
			return true
		}
	}
	for _, n := range a.Food {
		if n > 0 {
			return true
		}
	}
	return false
}

// ---------- Cứu đói (mục 7.4) ----------

func (a *Account) RescueEligible() bool {
	return a.Mana <= ManaHungryAt && a.Thoc < 5 && !a.hasEdibleOrSellable()
}

func (a *Account) StartRescue(now time.Time) (*RescueJob, error) {
	if a.Rescue != nil {
		return a.Rescue, nil // mỗi tài khoản một việc đang làm
	}
	if !a.RescueEligible() {
		return nil, errors.New("Việc cứu đói chỉ mở khi mana ≤ 20, ví dưới 5 thóc và không còn gì để ăn hoặc bán.")
	}
	a.Rescue = &RescueJob{ID: a.nextID("rescue"), StartAt: now}
	return a.Rescue, nil
}

func (a *Account) FinishRescue(jobID string, now time.Time, b Balance) error {
	if a.RescueDone[jobID] {
		return errors.New("Việc này đã nhận thưởng.")
	}
	if a.Rescue == nil || a.Rescue.ID != jobID {
		return errors.New("Không có việc đang làm.")
	}
	if now.Sub(a.Rescue.StartAt) < b.RescueDuration-300*time.Millisecond {
		return errors.New("Chưa dọn xong ổ rơm.")
	}
	a.RescueDone[jobID] = true
	a.Rescue = nil
	return a.addThoc(b.RescueReward, "rescue_job", jobID, now)
}

// ---------- Hướng dẫn ----------

func (a *Account) ClaimTutorial(now time.Time, b Balance) error {
	if a.TutorialClaimed {
		return errors.New("Thưởng hướng dẫn chỉ nhận một lần.")
	}
	if !a.tutorialReady() {
		return errors.New("Hoàn thành các bước hướng dẫn trước đã.")
	}
	a.TutorialClaimed = true
	// Thưởng hướng dẫn: thóc + 3 khẩu phần cám + 1 trứng thường (PLAN_LOGIC_GAME 2.3).
	a.Food["cam"] += 3
	a.Items["trung_thuong"]++
	return a.addThoc(b.TutorialReward, "tutorial", "tutorial", now)
}

// ---------- Thời trang (mục 8.3) ----------

func (a *Account) BuyCosmetic(id string, now time.Time) error {
	c := cosmeticByID(id)
	if c == nil {
		return errors.New("Món đồ không hợp lệ.")
	}
	if a.Owned[id] {
		return errors.New("Bạn đã sở hữu món này.") // không mua trùng
	}
	if c.Season != "" {
		if s := CurrentSeason(now); s == nil || s.ID != c.Season {
			return errors.New("Món này thuộc bộ sưu tập mùa, chỉ bán trong mùa.")
		}
	}
	if err := a.addThoc(-c.Cost, "buy_cosmetic", id, now); err != nil {
		return err
	}
	a.Owned[id] = true
	a.Equipped[c.Slot] = id
	return nil
}

func (a *Account) Equip(slot, id string) error {
	if !contains(Slots, slot) {
		return errors.New("Slot không hợp lệ.")
	}
	if id == "" {
		delete(a.Equipped, slot)
		return nil
	}
	c := cosmeticByID(id)
	if c == nil || c.Slot != slot {
		return errors.New("Món đồ không hợp slot.")
	}
	if !a.Owned[id] {
		return errors.New("Chỉ mặc được món đã sở hữu.")
	}
	a.Equipped[slot] = id
	return nil
}

func (a *Account) SaveOutfit(i int) error {
	if i < 0 || i > 2 {
		return errors.New("Ô phối đồ không hợp lệ.")
	}
	o := map[string]string{}
	for k, v := range a.Equipped {
		o[k] = v
	}
	a.Outfits[i] = o
	return nil
}

func (a *Account) LoadOutfit(i int) error {
	if i < 0 || i > 2 || a.Outfits[i] == nil {
		return errors.New("Ô phối đồ trống.")
	}
	eq := map[string]string{}
	for slot, id := range a.Outfits[i] {
		if a.Owned[id] { // chỉ trang bị vật phẩm còn sở hữu
			eq[slot] = id
		}
	}
	a.Equipped = eq
	return nil
}

// ---------- Khoe giun ----------

func (a *Account) Showcase(lootID string, now time.Time, b Balance) (*Loot, error) {
	var l *Loot
	for i := range a.Loots {
		if a.Loots[i].ID == lootID {
			l = &a.Loots[i]
		}
	}
	if l == nil {
		return nil, errors.New("Không tìm thấy lần đào này.")
	}
	if !wormByID(l.Worm).Rare {
		return nil, errors.New("Chỉ khoe giun từ Hiếm trở lên.")
	}
	if l.Shared {
		return nil, errors.New("Lần đào này đã khoe rồi.")
	}
	if wait := a.LastShowAt.Add(b.ShowcaseEvery).Sub(now); wait > 0 {
		return nil, fmt.Errorf("Mỗi người khoe tối đa 1 thẻ/%d giây. Chờ %.0f giây.", int(b.ShowcaseEvery.Seconds()), math.Ceil(wait.Seconds()))
	}
	l.Shared = true
	a.LastShowAt = now
	a.Stats.Shows++
	return l, nil
}

// ---------- Kiểm tra tên ----------

var blockedWords = []string{"admin", "moderator", "hypex", "đm", "dm ", "vcl", "cc ", "lồn", "cặc", "địt"}

func ValidateName(n string) (string, error) {
	n = strings.Join(strings.Fields(n), " ")
	l := len([]rune(n))
	if l < NameMinLen || l > NameMaxLen {
		return "", fmt.Errorf("Tên cần %d–%d ký tự.", NameMinLen, NameMaxLen)
	}
	low := strings.ToLower(n) + " "
	for _, w := range blockedWords {
		if strings.Contains(low, w) {
			return "", errors.New("Tên chứa từ không phù hợp, hãy chọn tên khác.")
		}
	}
	return n, nil
}

// ---------- Bản nhìn cho client ----------

type AccountView struct {
	ID              string              `json:"id"`
	Tag             string              `json:"tag"`
	Char            *Character          `json:"char"`
	Level           int                 `json:"level"`
	XP              int                 `json:"xp"`
	XPNeed          int                 `json:"xpNeed"`
	Thoc            int                 `json:"thoc"`
	Mana            float64             `json:"mana"`
	Stage           int                 `json:"stage"`
	ZeroElapsed     float64             `json:"zeroElapsed"`
	Worms           map[string]int      `json:"worms"`
	Food            map[string]int      `json:"food"`
	Collection      map[string]bool     `json:"collection"`
	Records         map[string]float64  `json:"records"`
	Owned           map[string]bool     `json:"owned"`
	Equipped        map[string]string   `json:"equipped"`
	Outfits         []map[string]string `json:"outfits"`
	DigReadyIn      float64             `json:"digReadyIn"`
	DailyDigs       int                 `json:"dailyDigs"`
	Daily           []Quest             `json:"daily"`
	Tutorial        map[string]bool     `json:"tutorial"`
	TutorialClaimed bool                `json:"tutorialClaimed"`
	RescueEligible  bool                `json:"rescueEligible"`
	Rescue          *RescueJob          `json:"rescue"`
	Stats           Stats               `json:"stats"`
	Loots           []Loot              `json:"loots"`
	BagCount        int                 `json:"bagCount"`
	BagCap          int                 `json:"bagCap"`
	Discover        map[string]bool     `json:"discover"`
	Titles          []string            `json:"titles"`
	Title           string              `json:"title"`
	Orders          []OrderView         `json:"orders"`
	Festival        int                 `json:"festival"`
	Eggs            int                 `json:"eggs"`
	EggsClaimed     map[int]bool        `json:"eggsClaimed"`
	Incubation      *Incubation         `json:"incubation"`
	HatchIn         float64             `json:"hatchIn"`
	Companions      []Companion         `json:"companions"`
	Following       string              `json:"following"`
	Booking         *Booking            `json:"booking"`
	TripsLeft       int                 `json:"tripsLeft"`
	OwnedVehicles   map[string]bool     `json:"ownedVehicles"`
	Reputation      int                 `json:"reputation"`
	Items           map[string]int      `json:"items"`
	StorageCount    int                 `json:"storageCount"`
	StorageCap      int                 `json:"storageCap"`
	LevelRewarded   map[int]bool        `json:"levelRewarded"`
	Coupons         map[string]float64  `json:"coupons"` // mã → giây còn hiệu lực
	BusTokens       int                 `json:"busTokens"`
	NextTokenIn     float64             `json:"nextTokenIn"`
	Prefs           Prefs               `json:"prefs"`
	CheckIn         CheckInView         `json:"checkIn"`
	Beds            map[string]*Plant   `json:"beds"`
	StarterGranted  bool                `json:"starterGranted"`
	RationReady     bool                `json:"rationReady"`
	CompanionCap    int                 `json:"companionCap"`
	HouseBuilt      bool                `json:"houseBuilt"`
	FishReadyIn     float64             `json:"fishReadyIn"`
	FishCaught      int                 `json:"fishCaught"`
}

func (a *Account) View(now time.Time, b Balance) AccountView {
	a.EnsureDaily(now)
	need := LevelXP(a.Level)
	if a.Level >= LevelCap {
		need = 0
	}
	return AccountView{
		ID: a.ID, Tag: PlayerTag(a.ID), Char: a.Char, Level: a.Level, XP: a.XP, XPNeed: need, Thoc: a.Thoc,
		Mana: math.Round(a.Mana*10) / 10, Stage: a.HungerStage(b), ZeroElapsed: a.ZeroElapsed,
		Worms: a.Worms, Food: a.Food, Collection: a.Collection, Records: a.Records, Owned: a.Owned,
		Equipped: a.Equipped, Outfits: a.Outfits, DigReadyIn: math.Max(0, a.DigReadyAt.Sub(now).Seconds()),
		DailyDigs: a.DailyDigs, Daily: a.Daily, Tutorial: a.Tutorial, TutorialClaimed: a.TutorialClaimed,
		RescueEligible: a.RescueEligible(), Rescue: a.Rescue, Stats: a.Stats, Loots: a.Loots,
		BagCount: a.BagCount(), BagCap: BagCapacity, Discover: a.Discover, Titles: a.Titles, Title: a.Title,
		Orders: a.OrderViews(now), Festival: a.Festival,
		Eggs: a.eggCount(), EggsClaimed: a.EggsClaimed, Incubation: a.Incubation, HatchIn: a.hatchIn(now), Companions: a.Companions,
		Following: a.Following, Booking: a.Booking, TripsLeft: a.TripsLeft(now, b), OwnedVehicles: a.OwnedVehicles, Reputation: a.Reputation,
		Items: a.Items, StorageCount: a.StorageCount(), StorageCap: StorageCap, LevelRewarded: a.LevelRewarded,
		Coupons: a.couponView(now), BusTokens: a.Tokens(now, b), NextTokenIn: a.NextTokenIn(now, b), Prefs: a.Prefs,
		CheckIn: a.CheckInView(now), Beds: a.Beds, StarterGranted: a.StarterGranted, CompanionCap: a.CompanionCap(), HouseBuilt: a.HouseBuilt,
		FishReadyIn: math.Max(0, a.FishReadyAt.Sub(now).Seconds()), FishCaught: a.FishCaught,
		RationReady: a.RationDay != DayKey(now) && a.Mana < 10 && a.Thoc < 5 && !a.hasEdibleOrSellable(),
	}
}

// Mã người chơi ngắn để phân biệt khi trùng tên (mục 5).
func PlayerTag(id string) string {
	if len(id) < 4 {
		return id
	}
	return strings.ToUpper(id[len(id)-4:])
}

// BagCount là tổng số giun đang giữ (thức ăn mua có ô riêng, không tính vào túi giun).
func (a *Account) BagCount() int {
	n := 0
	for _, c := range a.Worms {
		n += c
	}
	return n
}

func (a *Account) couponView(now time.Time) map[string]float64 {
	out := map[string]float64{}
	for id, exp := range a.Coupons {
		if now.Before(exp) {
			out[id] = math.Round(exp.Sub(now).Seconds())
		}
	}
	return out
}
