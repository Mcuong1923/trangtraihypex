package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Bảng xếp hạng tuần (PLAN_LOGIC_GAME 16): giun dài nhất, nhiều giun hiếm nhất, thắng PK (đấu người thật)
// nhiều nhất. Tuần theo giờ Việt Nam (ISO). Người đứng đầu mỗi bảng nhận danh hiệu khi tuần mới bắt đầu;
// không có thưởng thóc lớn để bảng chỉ là chuyện vui khoe nhau.

type WeeklyKind struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Unit  string `json:"unit"`
	Title string `json:"title"`
}

var WeeklyKinds = []WeeklyKind{
	{"longest", "Giun dài nhất", "cm", "vua_giun"},
	{"rare", "Nhiều giun hiếm nhất", "con", "san_hiem"},
	{"pk", "Thắng PK nhiều nhất", "trận", "vo_si_tuan"},
}

const WeeklyTop = 10

type WeekEntry struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Note  string  `json:"note,omitempty"` // bảng giun dài: tên loài
}

type WeeklyState struct {
	Week    string                 `json:"week"`
	Boards  map[string][]WeekEntry `json:"boards"`
	Last    string                 `json:"last"`    // tuần vừa chốt
	Winners map[string]WeekEntry   `json:"winners"` // bảng → người đứng đầu tuần vừa chốt
	// Owed: danh hiệu tuần đã thắng mà chưa trao (người thắng chưa vào game). Giữ qua nhiều tuần, không mất khi
	// người thắng vắng cả tuần kế tiếp (M11).
	Owed map[string][]string `json:"owed,omitempty"`
}

type Weekly struct {
	mu   sync.Mutex
	st   WeeklyState
	path string
}

func OpenWeekly(dir string) *Weekly {
	w := &Weekly{path: filepath.Join(dir, "weekly.json")}
	if b, err := os.ReadFile(w.path); err == nil {
		json.Unmarshal(b, &w.st)
	}
	if w.st.Boards == nil {
		w.st.Boards = map[string][]WeekEntry{}
	}
	return w
}

// roll chốt tuần cũ khi sang tuần mới (gọi dưới khoá).
func (w *Weekly) roll(now time.Time) {
	wk := weekKey(now)
	if w.st.Week == wk {
		return
	}
	if w.st.Week != "" {
		w.st.Last, w.st.Winners = w.st.Week, map[string]WeekEntry{}
		if w.st.Owed == nil {
			w.st.Owed = map[string][]string{}
		}
		for k, list := range w.st.Boards {
			if len(list) > 0 {
				w.st.Winners[k] = list[0]
				if !contains(w.st.Owed[list[0].ID], k) {
					w.st.Owed[list[0].ID] = append(w.st.Owed[list[0].ID], k)
				}
			}
		}
	}
	w.st.Week, w.st.Boards = wk, map[string][]WeekEntry{}
	_ = writeAtomic(w.path, w.st)
}

// Claimed xoá danh hiệu tuần đã trao xong khỏi hàng nợ.
func (w *Weekly) Claimed(id, kind string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !contains(w.st.Owed[id], kind) {
		return
	}
	l := w.st.Owed[id][:0]
	for _, k := range w.st.Owed[id] {
		if k != kind {
			l = append(l, k)
		}
	}
	if len(l) == 0 {
		delete(w.st.Owed, id)
	} else {
		w.st.Owed[id] = l
	}
	_ = writeAtomic(w.path, w.st)
}

// Submit ghi thành tích của một người: bảng giun dài giữ kỷ lục cao nhất, hai bảng còn lại là tổng của tuần.
func (w *Weekly) Submit(kind, id, name string, value float64, note string, now time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roll(now)
	list := w.st.Boards[kind]
	found := false
	for i := range list {
		if list[i].ID == id {
			found = true
			if value > list[i].Value {
				list[i] = WeekEntry{id, name, value, note}
			}
		}
	}
	if !found {
		list = append(list, WeekEntry{id, name, value, note})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].Value > list[j].Value })
	if len(list) > WeeklyTop {
		list = list[:WeeklyTop]
	}
	w.st.Boards[kind] = list
	_ = writeAtomic(w.path, w.st)
}

// TitlesFor: danh hiệu tuần người này đã thắng mà chưa nhận (tuần vừa chốt hoặc các tuần trước khi vắng mặt).
func (w *Weekly) TitlesFor(id string, now time.Time) []WeeklyKind {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roll(now)
	var out []WeeklyKind
	for _, k := range WeeklyKinds {
		e, won := w.st.Winners[k.ID]
		if (won && e.ID == id) || contains(w.st.Owed[id], k.ID) {
			out = append(out, k)
		}
	}
	return out
}

func (w *Weekly) View(now time.Time) map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.roll(now)
	boards := []map[string]any{}
	for _, k := range WeeklyKinds {
		entries := append([]WeekEntry{}, w.st.Boards[k.ID]...)
		boards = append(boards, map[string]any{"id": k.ID, "name": k.Name, "unit": k.Unit, "title": titleName(k.Title), "entries": entries})
	}
	winners := map[string]WeekEntry{}
	for k, e := range w.st.Winners {
		winners[k] = e
	}
	return map[string]any{"week": w.st.Week, "last": w.st.Last, "boards": boards, "winners": winners}
}

// ---------- Phía tài khoản ----------

type WeekStats struct {
	Key    string         `json:"key"`
	Rare   int            `json:"rare"`
	PKWins int            `json:"pkWins"`
	PKVs   map[string]int `json:"pkVs,omitempty"` // số trận thắng tính điểm trước từng đối thủ trong tuần
}

// PKPairPerWeek: thắng cùng một đối thủ quá số này trong tuần thì không tính thêm bảng tuần, điểm hội, danh hiệu
// (chặn hai tài khoản thông đồng cày bảng PK).
const PKPairPerWeek = 5

// countPKWin ghi một trận thắng tính điểm trước đối thủ opp; false khi cặp này đã đủ PKPairPerWeek trong tuần.
func (a *Account) countPKWin(opp string, now time.Time) bool {
	ws := a.week(now)
	if ws.PKVs == nil {
		ws.PKVs = map[string]int{}
	}
	if ws.PKVs[opp] >= PKPairPerWeek {
		return false
	}
	ws.PKVs[opp]++
	return true
}

func (a *Account) week(now time.Time) *WeekStats {
	if wk := weekKey(now); a.Week.Key != wk {
		a.Week = WeekStats{Key: wk}
	}
	return &a.Week
}

// weeklyDug ghi một con giun vừa kéo được lên bảng tuần.
func (h *Hub) weeklyDug(a *Account, w *Worm, size float64, now time.Time) {
	if h.weekly == nil || a.Char == nil {
		return
	}
	h.weekly.Submit("longest", a.ID, a.Char.Name, size, w.Name, now)
	if wormTier(w) >= 2 {
		ws := a.week(now)
		ws.Rare++
		h.weekly.Submit("rare", a.ID, a.Char.Name, float64(ws.Rare), "", now)
	}
}

func (h *Hub) weeklyPKWin(a *Account, now time.Time) {
	if h.weekly == nil || a.Char == nil {
		return
	}
	ws := a.week(now)
	ws.PKWins++
	h.weekly.Submit("pk", a.ID, a.Char.Name, float64(ws.PKWins), "", now)
}

// grantWeeklyTitles trao danh hiệu tuần trước khi người đứng đầu vào game.
func (r *Room) grantWeeklyTitles(p *Player, now time.Time) {
	if r.hub.weekly == nil {
		return
	}
	for _, k := range r.hub.weekly.TitlesFor(p.acc.ID, now) {
		if p.acc.hasTitle(k.Title) {
			r.hub.weekly.Claimed(p.acc.ID, k.ID)
			continue
		}
		p.acc.Titles = append(p.acc.Titles, k.Title)
		if err := r.hub.store.Save(p.acc); err == nil {
			r.hub.weekly.Claimed(p.acc.ID, k.ID)
			r.toast(p, "good", "🏆 Bạn đứng đầu bảng “"+k.Name+"” tuần trước: nhận danh hiệu "+titleName(k.Title)+"!")
		} else {
			p.acc.Titles = p.acc.Titles[:len(p.acc.Titles)-1] // lưu lỗi: để lần vào sau trao lại
		}
	}
}
