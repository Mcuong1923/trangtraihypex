package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Nhà riêng (mục 9.1): mỗi người có một tổ khởi đầu, nâng đất là nâng quyền sở hữu hiện có
// (không tranh lô), Bò Mộc xây chuồng theo mẫu, đặt đồ theo lưới, quyền vào riêng tư/bạn bè/công khai.

type Decor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emo   string `json:"emo"`
	Cost  int    `json:"cost"`
	Color string `json:"color"`
}

var Decors = []Decor{
	{"o_rom", "Ổ rơm", "🪺", 80, "#d9b65a"},
	{"mang_thoc", "Máng thóc", "🌾", 100, "#e7a73a"},
	{"luong_rau", "Luống rau cải", "🥬", 120, "#6fa03a"},
	{"luong_ca_rot", "Luống cà rốt", "🥕", 120, "#e9833a"},
	{"chau_hoa", "Chậu hoa", "🌷", 90, "#e2577a"},
	{"thung_go", "Thùng gỗ", "🛢️", 80, "#7b4f2d"},
	{"cay_xanh", "Cây xanh", "🌳", 150, "#3f7d3a"},
	{"den_long", "Đèn lồng", "🏮", 140, "#d64b3a"},
	{"ban_tiec", "Bàn tiệc", "🎂", 200, "#c99a5b"},
	{"bang_ten", "Bảng tên", "🪧", 100, "#9c6b36"},
}

func decorByID(id string) *Decor {
	for i := range Decors {
		if Decors[i].ID == id {
			return &Decors[i]
		}
	}
	return nil
}

type LandTier struct {
	Name  string  `json:"name"`
	Cost  int     `json:"cost"`
	Level int     `json:"level"`
	HalfX float64 `json:"halfX"`
	HalfZ float64 `json:"halfZ"`
}

// Bảng giá mục 8.2: tổ miễn phí, mảnh đất riêng 1.200, mở rộng 3.000; mở mua đất từ cấp 3 (mục 7.3).
var LandTiers = []LandTier{
	{"Tổ nhỏ", 0, 1, 4.5, 4},
	{"Mảnh đất riêng", 1200, 3, 6.5, 5.5},
	{"Đất mở rộng", 3000, 3, 8.5, 7},
}

const (
	CoopCost      = 500
	HomeCapacity  = 8
	HomeObjLimit  = 50
	HomeInviteTTL = 2 * time.Minute
)

var RoofColors = map[string]string{"do": "Mái đỏ", "xanh": "Mái xanh dương", "luc": "Mái xanh lá", "nau": "Mái gỗ nâu"}
var HomeAccess = []string{"private", "friends", "public"}

type PlacedObj struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	X    int    `json:"x"`
	Z    int    `json:"z"`
	Rot  int    `json:"rot"` // bội 90°
}

type CoopJob struct {
	Roof    string    `json:"roof"`
	State   string    `json:"state"` // building | done
	StartAt time.Time `json:"startAt"`
	DoneAt  time.Time `json:"doneAt"`
}

type Home struct {
	Owner     string         `json:"owner"`
	OwnerName string         `json:"ownerName"`
	Tier      int            `json:"tier"`
	Access    string         `json:"access"`
	Coop      *CoopJob       `json:"coop"`
	Objects   []PlacedObj    `json:"objects"`
	Storage   map[string]int `json:"storage"`
	Seq       int            `json:"seq"`
	PartyEnd  time.Time      `json:"partyEnd"` // bữa tiệc sân nhà đang mở tới lúc này (mục 11)
}

func (h *Home) Partying(now time.Time) bool { return now.Before(h.PartyEnd) }

func (h *Home) hasTable() bool {
	for _, o := range h.Objects {
		if o.Kind == "ban_tiec" {
			return true
		}
	}
	return false
}

// OpenParty mở phiên tiệc có giới hạn thời gian; cần đặt ít nhất một bàn tiệc.
func (h *Home) OpenParty(minutes int, now time.Time) error {
	if minutes != 30 && minutes != 60 {
		return errors.New("Chọn tiệc 30 hoặc 60 phút.")
	}
	if !h.hasTable() {
		return errors.New("Đặt một Bàn tiệc trong chuồng trước khi mở tiệc.")
	}
	h.PartyEnd = now.Add(time.Duration(minutes) * time.Minute)
	return nil
}

func NewHome(owner, name string) *Home {
	h := &Home{Owner: owner, OwnerName: name, Access: "friends", Storage: map[string]int{"chau_hoa": 1}}
	// Tổ nhỏ + một đồ trang trí miễn phí, không bán/chuyển nhượng (mục 8.2).
	h.Objects = []PlacedObj{{ID: "o1", Kind: "o_rom", X: 1, Z: -1}}
	h.Seq = 1
	return h
}

func (h *Home) tier() LandTier { return LandTiers[h.Tier] }

// Tick hoàn tất công trình theo giờ máy chủ, kể cả khi chủ offline (mục 9.1).
func (h *Home) Tick(now time.Time) bool {
	if h.Coop != nil && h.Coop.State == "building" && !now.Before(h.Coop.DoneAt) {
		h.Coop.State = "done"
		return true
	}
	return false
}

// Ô lưới hợp lệ: trong đất, không chặn lối vào (hành lang cổng trước), không lấn nền chuồng.
func (h *Home) cellOK(x, z int, ignore string) error {
	t := h.tier()
	if math.Abs(float64(x)) > t.HalfX-1 || math.Abs(float64(z)) > t.HalfZ-1 {
		return errors.New("Ô này nằm ngoài mảnh đất.")
	}
	if x >= -1 && x <= 1 && float64(z) >= t.HalfZ-3 {
		return errors.New("Không đặt đồ chắn lối vào.")
	}
	if h.Tier >= 1 {
		cx, cz := coopCell(t)
		if x >= cx-1 && x <= cx+1 && z >= cz-1 && z <= cz {
			return errors.New("Đây là nền chuồng, chọn ô khác.")
		}
	}
	for _, o := range h.Objects {
		if o.ID != ignore && o.X == x && o.Z == z {
			return errors.New("Ô này đã có đồ.")
		}
	}
	return nil
}

func coopCell(t LandTier) (int, int) {
	return -int(t.HalfX) + 2, -int(t.HalfZ) + 2
}

func (h *Home) BuyDecor(a *Account, id string, now time.Time) error {
	d := decorByID(id)
	if d == nil {
		return errors.New("Món trang trí không hợp lệ.")
	}
	if err := a.addThoc(-d.Cost, "buy_decor", id, now); err != nil {
		return err
	}
	h.Storage[id]++
	return nil
}

func (h *Home) Place(kind string, x, z, rot int) (PlacedObj, error) {
	if h.Storage[kind] <= 0 {
		return PlacedObj{}, errors.New("Kho không còn món này.")
	}
	if len(h.Objects) >= HomeObjLimit {
		return PlacedObj{}, fmt.Errorf("Tối đa %d đồ trang trí.", HomeObjLimit)
	}
	if err := h.cellOK(x, z, ""); err != nil {
		return PlacedObj{}, err
	}
	h.Seq++
	o := PlacedObj{ID: fmt.Sprintf("o%d", h.Seq), Kind: kind, X: x, Z: z, Rot: ((rot % 4) + 4) % 4}
	h.Storage[kind]--
	h.Objects = append(h.Objects, o)
	return o, nil
}

func (h *Home) obj(id string) int {
	for i, o := range h.Objects {
		if o.ID == id {
			return i
		}
	}
	return -1
}

func (h *Home) Move(id string, x, z, rot int) error {
	i := h.obj(id)
	if i < 0 {
		return errors.New("Không thấy món đồ.")
	}
	if err := h.cellOK(x, z, id); err != nil {
		return err
	}
	h.Objects[i].X, h.Objects[i].Z, h.Objects[i].Rot = x, z, ((rot%4)+4)%4
	return nil
}

func (h *Home) Pickup(id string) error {
	i := h.obj(id)
	if i < 0 {
		return errors.New("Không thấy món đồ.")
	}
	h.Storage[h.Objects[i].Kind]++
	h.Objects = append(h.Objects[:i], h.Objects[i+1:]...)
	return nil
}

// UpgradeLand nâng hạng đất; đồ còn hợp lệ giữ nguyên, đồ bị ảnh hưởng trả về kho, không mất (mục 9.1).
func (h *Home) UpgradeLand(a *Account, now time.Time) (int, error) {
	if h.Tier+1 >= len(LandTiers) {
		return 0, errors.New("Đã ở hạng đất cao nhất.")
	}
	next := LandTiers[h.Tier+1]
	if a.Level < next.Level {
		return 0, fmt.Errorf("Cần cấp %d để mua %s.", next.Level, next.Name)
	}
	if err := a.addThoc(-next.Cost, "buy_land", next.Name, now); err != nil {
		return 0, err
	}
	h.Tier++
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
	return moved, nil
}

func (h *Home) HireBuild(a *Account, roof string, now time.Time, dur time.Duration) error {
	if _, ok := RoofColors[roof]; !ok {
		return errors.New("Chọn màu mái hợp lệ.")
	}
	if h.Tier < 1 {
		return errors.New("Cần nâng lên Mảnh đất riêng trước khi xây chuồng.")
	}
	if h.Coop != nil {
		return errors.New("Chuồng đã xây hoặc đang xây.")
	}
	if err := a.addThoc(-CoopCost, "hire_build", roof, now); err != nil {
		return err
	}
	h.Coop = &CoopJob{Roof: roof, State: "building", StartAt: now, DoneAt: now.Add(dur)}
	return nil
}

// Zone dựng khu "home" cho phòng riêng: cổng về sảnh, Bò Mộc và bảng nhà.
func (h *Home) Zone() *Zone {
	t := h.tier()
	return &Zone{ID: "home", Title: "Chuồng của " + h.OwnerName, HalfX: t.HalfX - 0.4, HalfZ: t.HalfZ - 0.4,
		Spawns: []Vec{{0, t.HalfZ - 2.4}, {-1, t.HalfZ - 2.4}, {1, t.HalfZ - 2.4}, {0, t.HalfZ - 3.2}},
		Items: []Item{
			{ID: "ve_sanh", Kind: "home_exit", Label: "Về sảnh chuồng", Emo: "🏠", X: 0, Z: t.HalfZ - 0.6, R: 1.4},
			{ID: "bo_moc", Kind: "npc_cow", Label: "Bò Mộc", Emo: "🐮", X: t.HalfX - 1.6, Z: t.HalfZ - 1.8, R: 2.2},
			{ID: "bang_nha", Kind: "home_board", Label: "Bảng nhà", Emo: "🪧", X: -t.HalfX + 1.4, Z: t.HalfZ - 1.4, R: 2},
		}}
}

type HomeView struct {
	*Home
	TierInfo  LandTier  `json:"tierInfo"`
	NextTier  *LandTier `json:"nextTier"`
	CoopCell  [2]int    `json:"coopCell"`
	BuildLeft float64   `json:"buildLeft"`
	PartyLeft float64   `json:"partyLeft"`
}

func (h *Home) View(now time.Time) HomeView {
	v := HomeView{Home: h, TierInfo: h.tier()}
	if h.Tier+1 < len(LandTiers) {
		n := LandTiers[h.Tier+1]
		v.NextTier = &n
	}
	v.CoopCell[0], v.CoopCell[1] = coopCell(h.tier())
	if h.Coop != nil && h.Coop.State == "building" {
		v.BuildLeft = math.Max(0, h.Coop.DoneAt.Sub(now).Seconds())
	}
	v.PartyLeft = math.Max(0, h.PartyEnd.Sub(now).Seconds())
	return v
}

// ---------- Lưu trữ ----------

type HomeIndexEntry struct {
	Name     string    `json:"name"`
	Access   string    `json:"access"`
	Tier     int       `json:"tier"`
	PartyEnd time.Time `json:"partyEnd"`
}

type HomeStore struct {
	dir   string
	mu    sync.Mutex
	cache map[string]*Home
	index map[string]HomeIndexEntry
}

func OpenHomes(dir string) (*HomeStore, error) {
	hdir := filepath.Join(dir, "homes")
	if err := os.MkdirAll(hdir, 0o755); err != nil {
		return nil, err
	}
	s := &HomeStore{dir: hdir, cache: map[string]*Home{}, index: map[string]HomeIndexEntry{}}
	if b, err := os.ReadFile(filepath.Join(hdir, "index.json")); err == nil {
		json.Unmarshal(b, &s.index)
	}
	return s, nil
}

// Get trả về nhà (tạo tổ khởi đầu nếu chưa có). Chỉ goroutine phòng nhà đang mở được sửa nó.
func (s *HomeStore) Get(owner, name string) (*Home, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if h, ok := s.cache[owner]; ok {
		return h, nil
	}
	var h *Home
	if b, err := os.ReadFile(filepath.Join(s.dir, owner+".json")); err == nil {
		h = &Home{}
		if err := json.Unmarshal(b, h); err != nil {
			return nil, err
		}
		if h.Storage == nil {
			h.Storage = map[string]int{}
		}
	} else {
		h = NewHome(owner, name)
		if err := s.saveLocked(h); err != nil {
			return nil, err
		}
	}
	s.cache[owner] = h
	return h, nil
}

func (s *HomeStore) saveLocked(h *Home) error {
	if err := writeAtomic(filepath.Join(s.dir, h.Owner+".json"), h); err != nil {
		return err
	}
	s.index[h.Owner] = HomeIndexEntry{h.OwnerName, h.Access, h.Tier, h.PartyEnd}
	return writeAtomic(filepath.Join(s.dir, "index.json"), s.index)
}

func (s *HomeStore) Save(h *Home) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked(h)
}

func (s *HomeStore) Access(owner string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.index[owner]
	if !ok {
		return "friends", false
	}
	if time.Now().Before(e.PartyEnd) {
		return "public", true // đang mở tiệc: ai không bị chặn cũng ghé được
	}
	return e.Access, true
}

func (s *HomeStore) Public() map[string]HomeIndexEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]HomeIndexEntry{}
	for id, e := range s.index {
		if e.Access == "public" || time.Now().Before(e.PartyEnd) {
			out[id] = e
		}
	}
	return out
}
