package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Những bổ sung giúp cộng đồng có việc để làm (mục 11): bảng việc làng, sổ khám phá,
// danh hiệu và ngày hội góp thóc.

// ---------- Bảng việc làng ----------

type Order struct {
	ID     string         `json:"id"`
	NPC    string         `json:"npc"`
	Worm   string         `json:"worm,omitempty"`
	Item   string         `json:"item,omitempty"` // nhiệm vụ của NPC: cần vật phẩm trong kho thay vì giun
	N      int            `json:"n"`
	Reward int            `json:"reward"`
	Gift   map[string]int `json:"gift,omitempty"` // thưởng vật liệu, hạt giống hoặc trứng
	Line   string         `json:"line,omitempty"`
}

type OrderView struct {
	Order
	Text string `json:"text"`
	Done bool   `json:"done"`
}

// Đơn có thưởng cao hơn giá bán lẻ một chút để đáng công đi giao, nhưng mỗi đơn một lần/ngày nên có trần.
var OrderPool = []Order{
	{ID: "dat3", NPC: "Lợn Ủn", Worm: "dat", N: 3, Reward: 25},
	{ID: "dat6", NPC: "Gà Cụ Cục", Worm: "dat", N: 6, Reward: 45},
	{ID: "beo2", NPC: "Lợn Ủn", Worm: "beo", N: 2, Reward: 40},
	{ID: "beo4", NPC: "Bò Mộc", Worm: "beo", N: 4, Reward: 80},
	{ID: "sang1", NPC: "Chó Mực", Worm: "sang", N: 1, Reward: 70},
	{ID: "dat_beo", NPC: "Gà Cụ Cục", Worm: "beo", N: 1, Reward: 22},
}

// NPCQuests: nhiệm vụ của NPC (PLAN_LOGIC_GAME 16) đặt hàng vật liệu, nông sản hoặc giun, thưởng vật phẩm.
// Mỗi ngày bảng việc làng có thêm một nhiệm vụ như vậy, đổi theo ngày.
var NPCQuests = []Order{
	{ID: "q_rom10", NPC: "Bò Mộc", Item: "rom", N: 10, Reward: 15, Gift: map[string]int{"go": 4}, Line: "Bò đang lợp mái cho nhà bên, thiếu 10 bó rơm!"},
	{ID: "q_beo5", NPC: "Lợn Ủn", Worm: "beo", N: 5, Reward: 70, Gift: map[string]int{"hat_bi_ngo": 2}, Line: "Khách đặt mâm giun béo, Ủn cần gấp 5 con!"},
	{ID: "q_hoa1", NPC: "Vịt Bông", Item: "bo_hoa", N: 1, Reward: 10, Gift: map[string]int{"trung_thuong": 1}, Line: "Ai tặng Vịt một bó hoa, Vịt đổi cho quả trứng xinh!"},
	{ID: "q_cai3", NPC: "Gà Cụ Cục", Item: "cai", N: 3, Reward: 20, Gift: map[string]int{"hat_ca_rot": 2}, Line: "Cụ thèm bát canh cải, cháu hái giúp cụ 3 cây nhé."},
	{ID: "q_carot3", NPC: "Dê Thợ Máy", Item: "ca_rot", N: 3, Reward: 30, Gift: map[string]int{"dinh": 10}, Line: "Ăn cà rốt cho tinh mắt, vặn ốc mới chuẩn!"},
	{ID: "q_bingo1", NPC: "Chim Chíp", Item: "bi_ngo", N: 1, Reward: 35, Gift: map[string]int{"gach": 4}, Line: "Chíp muốn khắc đèn bí ngô cho bến xe!"},
	{ID: "q_go8", NPC: "Ông Chủ Đầu Tư", Item: "go", N: 8, Reward: 30, Gift: map[string]int{"ngoi": 3}, Line: "Dựng biển “NHÀ MẪU” cần 8 tấm gỗ, giá hời đây!"},
}

// OrdersFor chọn đơn của ngày theo giờ Việt Nam: luôn có một đơn giun đất cho người mới, hai đơn giun khác
// và một nhiệm vụ của NPC.
func OrdersFor(day string) []Order {
	h := fnv.New32a()
	h.Write([]byte(day))
	seed := int(h.Sum32())
	easy := []Order{OrderPool[0], OrderPool[1]}
	out := []Order{easy[seed%2]}
	rest := OrderPool[2:]
	a := (seed / 2) % len(rest)
	b := (a + 1 + (seed/7)%(len(rest)-1)) % len(rest)
	return append(out, rest[a], rest[b], NPCQuests[(seed/11)%len(NPCQuests)])
}

func (o Order) need() (string, string) {
	if o.Item != "" {
		return o.Item, itemByID(o.Item).Name
	}
	return o.Worm, wormByID(o.Worm).Name
}

func (a *Account) OrderViews(now time.Time) []OrderView {
	day := DayKey(now)
	var out []OrderView
	for _, o := range OrdersFor(day) {
		_, name := o.need()
		out = append(out, OrderView{Order: o, Text: fmt.Sprintf("%s cần %d %s", o.NPC, o.N, name), Done: a.OrdersDone[day+":"+o.ID]})
	}
	return out
}

func (a *Account) DeliverOrder(id string, now time.Time) (Order, []string, error) {
	day := DayKey(now)
	for _, o := range OrdersFor(day) {
		if o.ID != id {
			continue
		}
		if a.OrdersDone[day+":"+o.ID] {
			return o, nil, errors.New("Đơn này hôm nay đã giao rồi.")
		}
		id, name := o.need()
		if o.Item != "" {
			if a.Items[id] < o.N {
				return o, nil, fmt.Errorf("Cần %d %s, kho đang có %d.", o.N, name, a.Items[id])
			}
			a.Items[id] -= o.N
			if err := a.addItems(o.Gift); err != nil {
				a.Items[id] += o.N
				return o, nil, err
			}
		} else {
			if a.Worms[id] < o.N {
				return o, nil, fmt.Errorf("Cần %d %s, túi đang có %d.", o.N, name, a.Worms[id])
			}
			if err := a.addItems(o.Gift); err != nil {
				return o, nil, err
			}
			a.Worms[id] -= o.N
		}
		a.OrdersDone[day+":"+o.ID] = true
		// Dọn khoá ngày cũ để bản lưu không phình.
		for k := range a.OrdersDone {
			if len(k) < len(day) || k[:len(day)] != day {
				delete(a.OrdersDone, k)
			}
		}
		_ = a.addThoc(o.Reward, "village_order", day+":"+o.ID, now)
		return o, a.CheckTitles(), nil
	}
	return Order{}, nil, errors.New("Đơn không có trên bảng hôm nay.")
}

// ---------- Sổ khám phá & danh hiệu ----------

type TitleDef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Hint string `json:"hint"`
}

var TitleDefs = []TitleDef{
	{"tham_hiem", "Nhà thám hiểm", "Đi qua sảnh, vườn giun, chợ, sân tỷ thí và chuồng riêng"},
	{"muon_loai", "Bạn của muôn loài", "Gặp Gà Cụ Cục, Lợn Ủn, Chó Mực và Bò Mộc"},
	{"tho_dao", "Thợ đào lành nghề", "Đào 50 lượt có thưởng"},
	{"suu_tam", "Nhà sưu tầm", "Có đủ 5 loài giun trong sổ"},
	{"cao_thu", "Cao thủ PK", "Thắng 10 trận PK"},
	{"hao_tam", "Nhà hảo tâm", "Góp từ 100 thóc cho ngày hội"},
	{"lao_lang", "Gà Lão Làng", "Đạt cấp 10 (quà lên cấp)"},
	{"vua_giun", "Vua Giun Dài", "Đứng đầu bảng “Giun dài nhất” của một tuần"},
	{"san_hiem", "Thợ Săn Giun Hiếm", "Đứng đầu bảng “Nhiều giun hiếm nhất” của một tuần"},
	{"vo_si_tuan", "Võ Sĩ Của Tuần", "Đứng đầu bảng “Thắng PK nhiều nhất” của một tuần"},
}

var discoverZones = []string{"lobby", "garden", "market", "arena", "home"}
var discoverNPCs = []string{"ga_cu_cuc", "lon_un", "cho_muc", "bo_moc"}

// Discovered ghi dấu một khu/NPC; trả về ghi chú khi mở danh hiệu mới.
func (a *Account) Discovered(key string) []string {
	if a.Discover[key] {
		return nil
	}
	a.Discover[key] = true
	return a.CheckTitles()
}

func (a *Account) hasTitle(id string) bool { return contains(a.Titles, id) }

func (a *Account) CheckTitles() []string {
	all := func(prefix string, keys []string) bool {
		for _, k := range keys {
			if !a.Discover[prefix+k] {
				return false
			}
		}
		return true
	}
	species := 0
	for _, w := range Worms {
		if a.Collection[w.ID] {
			species++
		}
	}
	got := map[string]bool{
		"tham_hiem": all("zone:", discoverZones),
		"muon_loai": all("npc:", discoverNPCs),
		"tho_dao":   a.Stats.Dug >= 50,
		"suu_tam":   species >= len(Worms),
		"cao_thu":   a.Stats.PKWins >= 10,
		"hao_tam":   a.Festival >= 100,
		"lao_lang":  a.Level >= LevelCap,
	}
	var notes []string
	for _, t := range TitleDefs {
		if got[t.ID] && !a.hasTitle(t.ID) {
			a.Titles = append(a.Titles, t.ID)
			notes = append(notes, "🏅 Mở danh hiệu mới: "+t.Name)
		}
	}
	return notes
}

func (a *Account) SetTitle(id string) error {
	if id != "" && !a.hasTitle(id) {
		return errors.New("Bạn chưa mở danh hiệu này.")
	}
	a.Title = id
	return nil
}

func titleName(id string) string {
	for _, t := range TitleDefs {
		if t.ID == id {
			return t.Name
		}
	}
	return ""
}

// ---------- Ngày hội góp thóc ----------

// Festival là mục tiêu chung toàn cộng đồng: có trần, góp tự nguyện, thưởng chủ yếu thẩm mỹ.
type FestivalState struct {
	Name     string `json:"name"`
	Goal     int    `json:"goal"`
	Total    int    `json:"total"`
	Unlocked bool   `json:"unlocked"`
}

type Festival struct {
	mu   sync.Mutex
	st   FestivalState
	path string
}

var FestivalAmounts = []int{10, 50, 100}

func OpenFestival(dir string) (*Festival, error) {
	f := &Festival{st: FestivalState{Name: "Hội đèn lồng mùa thu", Goal: 5000}, path: filepath.Join(dir, "festival.json")}
	if b, err := os.ReadFile(f.path); err == nil {
		if err := json.Unmarshal(b, &f.st); err != nil {
			return nil, err
		}
	}
	return f, nil
}

// Add cộng phần góp nếu chưa vượt trần; trả về trạng thái sau khi cộng.
func (f *Festival) Add(n int) (FestivalState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.st.Unlocked {
		return f.st, errors.New("Ngày hội đã đủ mục tiêu, cảm ơn bạn!")
	}
	if left := f.st.Goal - f.st.Total; n > left {
		return f.st, fmt.Errorf("Chỉ còn thiếu %d thóc để đạt mục tiêu.", left)
	}
	f.st.Total += n
	f.st.Unlocked = f.st.Total >= f.st.Goal
	if err := writeAtomic(f.path, f.st); err != nil {
		f.st.Total -= n
		f.st.Unlocked = false
		return f.st, err
	}
	return f.st, nil
}

// Undo hoàn phần góp khi giao dịch phía tài khoản không lưu được.
func (f *Festival) Undo(n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.st.Total -= n
	f.st.Unlocked = f.st.Total >= f.st.Goal
	_ = writeAtomic(f.path, f.st)
}

func (f *Festival) Snapshot() FestivalState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.st
}
