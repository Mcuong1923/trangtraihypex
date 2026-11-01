package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// Quản trị và vận hành (mục 19): xem người chơi/phòng, xử lý báo cáo, mute/ban có thời hạn và lý do,
// truy vết giao dịch, thông báo, bảo trì. Quyền kiểm duyệt (mod) tách với quyền xem kinh tế (econ);
// mọi thao tác ghi nhật ký quản trị.

type ModEntry struct {
	MutedUntil  time.Time `json:"mutedUntil"`
	BannedUntil time.Time `json:"bannedUntil"`
	Reason      string    `json:"reason"`
	By          string    `json:"by"`
}

type Moderation struct {
	mu      sync.Mutex
	path    string
	entries map[string]ModEntry
	auditF  *os.File
}

func OpenModeration(dir string) (*Moderation, error) {
	m := &Moderation{path: filepath.Join(dir, "moderation.json"), entries: map[string]ModEntry{}}
	if b, err := os.ReadFile(m.path); err == nil {
		json.Unmarshal(b, &m.entries)
	}
	f, err := os.OpenFile(filepath.Join(dir, "admin_audit.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	m.auditF = f
	return m, nil
}

func (m *Moderation) get(id string) ModEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries[id]
}

// Muted trả về thời điểm hết khoá chat nếu đang bị khoá.
func (m *Moderation) Muted(id string) (time.Time, string, bool) {
	e := m.get(id)
	return e.MutedUntil, e.Reason, time.Now().Before(e.MutedUntil)
}

func (m *Moderation) Banned(id string) (time.Time, string, bool) {
	e := m.get(id)
	return e.BannedUntil, e.Reason, time.Now().Before(e.BannedUntil)
}

func (m *Moderation) Set(id string, f func(*ModEntry)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[id]
	f(&e)
	m.entries[id] = e
	return writeAtomic(m.path, m.entries)
}

type AuditLine struct {
	At     time.Time `json:"at"`
	Staff  string    `json:"staff"`
	Action string    `json:"action"`
	Target string    `json:"target,omitempty"`
	Detail string    `json:"detail,omitempty"`
}

func (m *Moderation) Audit(l AuditLine) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l.At = time.Now()
	json.NewEncoder(m.auditF).Encode(l)
}

// tail đọc n dòng JSON cuối của một file append-only.
func tail[T any](path string, n int, keep func(T) bool) []T {
	f, err := os.Open(path)
	if err != nil {
		return []T{}
	}
	defer f.Close()
	out := []T{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		var v T
		if json.Unmarshal(sc.Bytes(), &v) == nil && (keep == nil || keep(v)) {
			out = append(out, v)
			if len(out) > n*2 {
				out = out[len(out)-n:]
			}
		}
	}
	if len(out) > n {
		out = out[len(out)-n:]
	}
	return out
}

// ---------- HTTP quản trị ----------

type staffCtx struct {
	ID, Username, Role string
}

func (s staffCtx) can(perm string) bool {
	return s.Role == "admin" || s.Role == perm
}

// ParseStaff đọc cờ "-staff an:mod,binh:econ,chu:admin".
func ParseStaff(spec string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(spec, ",") {
		if u, r, ok := strings.Cut(strings.TrimSpace(part), ":"); ok && u != "" && contains([]string{"mod", "econ", "admin"}, r) {
			out[strings.ToLower(u)] = r
		}
	}
	return out
}

func (h *Hub) staffOf(r *http.Request) (staffCtx, bool) {
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	id, ok := h.store.AccountIDForToken(tok)
	if !ok {
		return staffCtx{}, false
	}
	acc, err := h.store.Account(id)
	if err != nil {
		return staffCtx{}, false
	}
	role, ok := h.staff[acc.Username]
	return staffCtx{id, acc.Username, role}, ok
}

func (h *Hub) adminAPI(perm string, fn func(w http.ResponseWriter, r *http.Request, st staffCtx)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, ok := h.staffOf(r)
		if !ok {
			writeJSON(w, 403, map[string]any{"err": "Tài khoản không có quyền quản trị."})
			return
		}
		if perm != "" && !st.can(perm) {
			writeJSON(w, 403, map[string]any{"err": "Vai trò của bạn không có quyền này."})
			return
		}
		fn(w, r, st)
	}
}

func decode(r *http.Request, v any) error {
	return json.NewDecoder(http.MaxBytesReader(nil, r.Body, 8192)).Decode(v)
}

func (h *Hub) RegisterAdmin(mux *http.ServeMux) {
	mux.HandleFunc("/api/admin/me", h.adminAPI("", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		writeJSON(w, 200, map[string]any{"username": st.Username, "role": st.Role})
	}))

	mux.HandleFunc("/api/admin/overview", h.adminAPI("", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		type pl struct {
			Name, Code, Zone string
			Level            int
			Gone             bool
		}
		type rm struct {
			ID, Name, Kind string
			Count          int
			Players        []pl
		}
		h.mu.Lock()
		rooms := append([]*Room(nil), h.rooms...)
		for _, r := range h.homes {
			rooms = append(rooms, r)
		}
		online := len(h.sessions)
		maint := h.maintenance
		h.mu.Unlock()
		var out []rm
		for _, r := range rooms {
			x := rm{ID: r.ID, Name: r.Name, Kind: r.Kind, Count: r.Count()}
			r.DoWait(func() {
				for _, p := range r.players {
					x.Players = append(x.Players, pl{p.acc.Char.Name, h.store.Social.Code(p.acc.ID), p.zone, p.acc.Level, !p.goneAt.IsZero()})
				}
			})
			out = append(out, x)
		}
		writeJSON(w, 200, map[string]any{"online": online, "rooms": out, "maintenance": maint, "metrics": h.Metrics(), "festival": h.festival.Snapshot(), "balance": h.bal.Name})
	}))

	mux.HandleFunc("/api/admin/reports", h.adminAPI("mod", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		writeJSON(w, 200, map[string]any{"reports": h.store.Social.Reports()})
	}))

	mux.HandleFunc("/api/admin/reports/resolve", h.adminAPI("mod", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		var b struct {
			ID, Action, Note string
			Minutes          int
		}
		if decode(r, &b) != nil {
			writeJSON(w, 400, map[string]any{"err": "Dữ liệu không hợp lệ."})
			return
		}
		rep, err := h.store.Social.ResolveReport(b.ID, b.Action, st.Username, b.Note)
		if err == nil && (b.Action == "mute" || b.Action == "ban") {
			err = h.moderate(st, rep.Target, b.Action, b.Minutes, rep.Reason+": "+b.Note)
		}
		if err != nil {
			writeJSON(w, 400, map[string]any{"err": err.Error()})
			return
		}
		h.store.Mod.Audit(AuditLine{Staff: st.Username, Action: "report_" + b.Action, Target: rep.Target, Detail: b.ID + " " + b.Note})
		writeJSON(w, 200, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/admin/moderate", h.adminAPI("mod", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		var b struct {
			Code, Action, Reason string
			Minutes              int
		}
		if decode(r, &b) != nil {
			writeJSON(w, 400, map[string]any{"err": "Dữ liệu không hợp lệ."})
			return
		}
		id, ok := h.store.Social.Lookup(b.Code)
		if !ok {
			writeJSON(w, 400, map[string]any{"err": "Không tìm thấy mã người chơi."})
			return
		}
		if err := h.moderate(st, id, b.Action, b.Minutes, b.Reason); err != nil {
			writeJSON(w, 400, map[string]any{"err": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/admin/announce", h.adminAPI("mod", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		var b struct{ Text string }
		if decode(r, &b) != nil || strings.TrimSpace(b.Text) == "" {
			writeJSON(w, 400, map[string]any{"err": "Nội dung trống."})
			return
		}
		h.Announce("📢 " + strings.TrimSpace(b.Text))
		h.store.Mod.Audit(AuditLine{Staff: st.Username, Action: "announce", Detail: b.Text})
		writeJSON(w, 200, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/admin/maintenance", h.adminAPI("admin", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		var b struct {
			On   bool
			Text string
		}
		if decode(r, &b) != nil {
			writeJSON(w, 400, map[string]any{"err": "Dữ liệu không hợp lệ."})
			return
		}
		h.mu.Lock()
		h.maintenance = b.On
		h.maintText = b.Text
		h.mu.Unlock()
		if b.On {
			h.Announce("🛠️ Sắp bảo trì: " + b.Text + " — người chơi đang trong game vẫn giữ tài sản, chỉ tạm dừng vào mới.")
		}
		h.store.Mod.Audit(AuditLine{Staff: st.Username, Action: fmt.Sprintf("maintenance=%v", b.On), Detail: b.Text})
		writeJSON(w, 200, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/admin/ledger", h.adminAPI("econ", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		code := r.URL.Query().Get("code")
		id, ok := h.store.Social.Lookup(code)
		if !ok {
			writeJSON(w, 400, map[string]any{"err": "Không tìm thấy mã người chơi."})
			return
		}
		lines := tail[LedgerEntry](filepath.Join(h.store.dir, "ledger.jsonl"), 200, func(e LedgerEntry) bool { return e.Account == id })
		h.store.Mod.Audit(AuditLine{Staff: st.Username, Action: "view_ledger", Target: id})
		writeJSON(w, 200, map[string]any{"name": h.store.Social.Name(id), "entries": lines})
	}))

	mux.HandleFunc("/api/admin/audit", h.adminAPI("admin", func(w http.ResponseWriter, r *http.Request, st staffCtx) {
		writeJSON(w, 200, map[string]any{"lines": tail[AuditLine](filepath.Join(h.store.dir, "admin_audit.jsonl"), 200, nil)})
	}))
}

// moderate áp dụng mute/ban/kick có thời hạn và lý do; người đang online nhận hiệu lực ngay.
func (h *Hub) moderate(st staffCtx, id, action string, minutes int, reason string) error {
	if minutes <= 0 {
		minutes = 60
	}
	if strings.TrimSpace(reason) == "" && action != "unmute" && action != "unban" {
		return errors.New("Cần ghi lý do.")
	}
	until := time.Now().Add(time.Duration(minutes) * time.Minute)
	var err error
	switch action {
	case "mute":
		err = h.store.Mod.Set(id, func(e *ModEntry) { e.MutedUntil, e.Reason, e.By = until, reason, st.Username })
		if s := h.sessionOf(id); s != nil {
			s.Send(map[string]any{"t": "toast", "kind": "bad", "text": fmt.Sprintf("Bạn bị khoá chat đến %s. Lý do: %s", until.In(vnZone).Format("15:04 02/01"), reason)})
		}
	case "unmute":
		err = h.store.Mod.Set(id, func(e *ModEntry) { e.MutedUntil = time.Time{} })
	case "ban":
		err = h.store.Mod.Set(id, func(e *ModEntry) { e.BannedUntil, e.Reason, e.By = until, reason, st.Username })
		if err == nil {
			h.kick(id, fmt.Sprintf("Tài khoản bị tạm khoá đến %s. Lý do: %s", until.In(vnZone).Format("15:04 02/01"), reason))
		}
	case "unban":
		err = h.store.Mod.Set(id, func(e *ModEntry) { e.BannedUntil = time.Time{} })
	case "kick":
		h.kick(id, "Bạn đã bị mời ra khỏi game. Lý do: "+reason)
	default:
		return errors.New("Hành động không hợp lệ.")
	}
	if err == nil {
		h.store.Mod.Audit(AuditLine{Staff: st.Username, Action: action, Target: id, Detail: fmt.Sprintf("%d phút · %s", minutes, reason)})
	}
	return err
}

// kick đưa phiên ra khỏi phòng ngay, không giữ chỗ nối lại.
func (h *Hub) kick(id, reason string) {
	h.mu.Lock()
	s := h.sessions[id]
	delete(h.sessions, id)
	h.mu.Unlock()
	if s == nil {
		return
	}
	s.kicked.Store(true)
	s.Send(map[string]any{"t": "kicked", "reason": reason, "final": true})
	if r := s.room.Swap(nil); r != nil {
		r.DoWait(func() { r.remove(s) })
	}
	time.AfterFunc(300*time.Millisecond, s.Close)
}

// Announce gửi thông báo hệ thống tới mọi phòng đang mở.
func (h *Hub) Announce(text string) {
	h.mu.Lock()
	rooms := append([]*Room(nil), h.rooms...)
	for _, r := range h.homes {
		rooms = append(rooms, r)
	}
	h.mu.Unlock()
	for _, r := range rooms {
		r.Do(func() { r.broadcastRoom(nil, map[string]any{"t": "chat", "kind": "system", "text": text}) })
	}
}

// BroadcastAll gửi một tin tới mọi phòng (dùng từ goroutine riêng, không từ goroutine phòng).
func (h *Hub) BroadcastAll(v map[string]any) {
	h.mu.Lock()
	rooms := append([]*Room(nil), h.rooms...)
	for _, r := range h.homes {
		rooms = append(rooms, r)
	}
	h.mu.Unlock()
	for _, r := range rooms {
		r.Do(func() { r.broadcastRoom(nil, v) })
	}
}

// ---------- Số đo vận hành ----------

type Metrics struct {
	Online    int     `json:"online"`
	Rooms     int     `json:"rooms"`
	TickP50ms float64 `json:"tickP50ms"`
	TickP95ms float64 `json:"tickP95ms"`
	TickMaxms float64 `json:"tickMaxms"`
	Drops     int64   `json:"sendDrops"`
	HeapMB    float64 `json:"heapMB"`
	Goroutine int     `json:"goroutines"`
}

func (h *Hub) Metrics() Metrics {
	h.mu.Lock()
	rooms := append([]*Room(nil), h.rooms...)
	for _, r := range h.homes {
		rooms = append(rooms, r)
	}
	m := Metrics{Online: len(h.sessions), Rooms: len(rooms)}
	h.mu.Unlock()
	var all []float64
	for _, r := range rooms {
		all = append(all, r.TickSamples()...)
	}
	if len(all) > 0 {
		sort.Float64s(all)
		m.TickP50ms = all[len(all)/2]
		m.TickP95ms = all[len(all)*95/100]
		m.TickMaxms = all[len(all)-1]
	}
	m.Drops = sendDrops.Load()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.HeapMB = float64(ms.HeapAlloc) / (1 << 20)
	m.Goroutine = runtime.NumGoroutine()
	return m
}
