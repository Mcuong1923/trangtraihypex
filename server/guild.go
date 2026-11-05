package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Hội Gà (mục 13, V1.2). Dữ liệu hội nằm ngoài Account vì nhiều thành viên ở nhiều phòng cùng sửa;
// mọi thao tác đi qua khoá của GuildStore. Phần chạm ví (lập hội, góp quỹ) do phòng giữ tài khoản làm.

type GuildMember struct {
	Name        string    `json:"name"`
	Role        string    `json:"role"` // owner | vice | member
	JoinedAt    time.Time `json:"joinedAt"`
	Contributed int       `json:"contributed"`
	LastSeen    time.Time `json:"lastSeen"`
}

type GuildPost struct {
	By   string    `json:"by"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type GuildLedger struct {
	At     time.Time `json:"at"`
	Who    string    `json:"who"`
	Delta  int       `json:"delta"`
	Reason string    `json:"reason"`
}

type Guild struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Emblem    string                  `json:"emblem"`
	Color     string                  `json:"color"`
	Owner     string                  `json:"owner"`
	JoinMode  string                  `json:"joinMode"` // open | approval
	Members   map[string]*GuildMember `json:"members"`
	Requests  map[string]string       `json:"requests"` // id → tên
	Invites   map[string]time.Time    `json:"invites"`
	Fund      int                     `json:"fund"`
	Ledger    []GuildLedger           `json:"ledger"`
	Board     []GuildPost             `json:"board"`
	Projects  map[string]bool         `json:"projects"`
	Transfer  string                  `json:"transfer"` // người được đề nghị nhận chức chủ hội
	Week      string                  `json:"week"`
	WeekScore map[string]int          `json:"weekScore"`
	CreatedAt time.Time               `json:"createdAt"`
}

type GuildProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Cost int    `json:"cost"`
	Perk string `json:"perk"`
}

// Quỹ hội chỉ chi cho công trình đã định nghĩa (mục 13).
var GuildProjects = []GuildProject{
	{"co_hoi", "Cờ hội", 2000, "Treo cờ màu hội ở Nhà Hội, tên thành viên mang màu hội"},
	{"den_hoi", "Đèn lồng hội", 3000, "Thắp đèn lồng quanh sân họp"},
	{"nha_hoi_2", "Nâng Nhà Hội cấp 2", 5000, "Thêm sân khấu, ghế và mái che cho sân họp"},
}

var GuildEmblems = []string{"🐔", "🌾", "🪱", "🥚", "🌻", "⭐", "🏮", "🍀"}
var GuildColors = []string{"#d64b3a", "#3f86c9", "#4e9a4e", "#e0a53a", "#8a6fc0", "#3b3530"}

type GuildStore struct {
	mu     sync.Mutex
	path   string
	Guilds map[string]*Guild `json:"guilds"`
	member map[string]string // account → guild
}

func OpenGuilds(dir string) (*GuildStore, error) {
	g := &GuildStore{path: filepath.Join(dir, "guilds.json"), Guilds: map[string]*Guild{}}
	if b, err := os.ReadFile(g.path); err == nil {
		if err := json.Unmarshal(b, g); err != nil {
			return nil, err
		}
	}
	g.member = map[string]string{}
	for id, gd := range g.Guilds {
		for m := range gd.Members {
			g.member[m] = id
		}
	}
	return g, nil
}

func (g *GuildStore) saveLocked() error { return writeAtomic(g.path, g) }

func (g *GuildStore) of(acc string) *Guild {
	if id := g.member[acc]; id != "" {
		return g.Guilds[id]
	}
	return nil
}

func (g *GuildStore) GuildOf(acc string) (string, string, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if gd := g.of(acc); gd != nil {
		return gd.ID, gd.Name, gd.Color
	}
	return "", "", ""
}

func (g *GuildStore) Members(id string) []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	if gd := g.Guilds[id]; gd != nil {
		for m := range gd.Members {
			out = append(out, m)
		}
	}
	return out
}

func (gd *Guild) role(acc string) string {
	if m := gd.Members[acc]; m != nil {
		return m.Role
	}
	return ""
}

func (gd *Guild) ledger(who string, delta int, reason string) {
	gd.Ledger = append(gd.Ledger, GuildLedger{time.Now(), who, delta, reason})
	if len(gd.Ledger) > 200 {
		gd.Ledger = gd.Ledger[len(gd.Ledger)-200:]
	}
}

func (g *GuildStore) nameTaken(name string) bool {
	for _, gd := range g.Guilds {
		if strings.EqualFold(gd.Name, name) {
			return true
		}
	}
	return false
}

// Create lập hội sau khi phòng đã trừ phí lập hội khỏi ví.
func (g *GuildStore) Create(owner, ownerName, name, emblem, color string) (*Guild, error) {
	n, err := ValidateName(name)
	if err != nil {
		return nil, err
	}
	if !contains(GuildEmblems, emblem) {
		emblem = GuildEmblems[0]
	}
	if !contains(GuildColors, color) {
		color = GuildColors[0]
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.of(owner) != nil {
		return nil, errors.New("Bạn đang ở trong một hội.")
	}
	if g.nameTaken(n) {
		return nil, errors.New("Tên hội đã có, chọn tên khác.")
	}
	now := time.Now()
	gd := &Guild{ID: "hoi" + randHex(4), Name: n, Emblem: emblem, Color: color, Owner: owner, JoinMode: "approval",
		Members:  map[string]*GuildMember{owner: {Name: ownerName, Role: "owner", JoinedAt: now, LastSeen: now}},
		Requests: map[string]string{}, Invites: map[string]time.Time{}, Projects: map[string]bool{}, WeekScore: map[string]int{}, CreatedAt: now}
	g.Guilds[gd.ID] = gd
	g.member[owner] = gd.ID
	if err := g.saveLocked(); err != nil {
		delete(g.Guilds, gd.ID)
		delete(g.member, owner)
		return nil, err
	}
	return gd, nil
}

func (g *GuildStore) addMemberLocked(gd *Guild, id, name string) error {
	if len(gd.Members) >= GuildMax {
		return fmt.Errorf("Hội đã đủ %d thành viên.", GuildMax)
	}
	if g.of(id) != nil {
		return errors.New("Người này đã ở trong một hội.")
	}
	gd.Members[id] = &GuildMember{Name: name, Role: "member", JoinedAt: time.Now(), LastSeen: time.Now()}
	delete(gd.Requests, id)
	delete(gd.Invites, id)
	g.member[id] = gd.ID
	return nil
}

// Do chạy một thao tác có kiểm quyền trên hội của actor, rồi lưu.
func (g *GuildStore) Do(actor string, f func(gd *Guild) error) (*Guild, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.of(actor)
	if gd == nil {
		return nil, errors.New("Bạn chưa ở trong hội nào.")
	}
	snap, _ := json.Marshal(gd)
	if err := f(gd); err != nil {
		g.rollbackLocked(gd.ID, snap)
		return nil, err
	}
	if err := g.saveLocked(); err != nil {
		g.rollbackLocked(gd.ID, snap) // không để quỹ/thành viên đổi trong bộ nhớ rồi lần lưu sau ghi lại (M6)
		return nil, err
	}
	return gd, nil
}

// rollbackLocked trả hội id về bản chụp trước thao tác và dựng lại bảng thành viên → hội.
func (g *GuildStore) rollbackLocked(id string, snap []byte) {
	var old Guild
	if err := json.Unmarshal(snap, &old); err != nil {
		log.Printf("guild rollback %s: %v", id, err)
		return
	}
	g.Guilds[id] = &old
	g.member = map[string]string{}
	for gid, gd := range g.Guilds {
		for m := range gd.Members {
			g.member[m] = gid
		}
	}
}

func (g *GuildStore) Join(id, name, guildID string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.Guilds[guildID]
	if gd == nil {
		return "", errors.New("Không tìm thấy hội.")
	}
	if g.of(id) != nil {
		return "", errors.New("Bạn đang ở trong một hội.")
	}
	if _, invited := gd.Invites[id]; invited || gd.JoinMode == "open" {
		if err := g.addMemberLocked(gd, id, name); err != nil {
			return "", err
		}
		return "joined", g.saveLocked()
	}
	gd.Requests[id] = name
	return "requested", g.saveLocked()
}

func (g *GuildStore) Leave(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.of(id)
	if gd == nil {
		return errors.New("Bạn chưa ở trong hội nào.")
	}
	if gd.Owner == id && len(gd.Members) > 1 {
		return errors.New("Chủ hội cần chuyển quyền trước khi rời hội.")
	}
	delete(gd.Members, id)
	delete(g.member, id)
	if len(gd.Members) == 0 {
		delete(g.Guilds, gd.ID) // hội không còn ai thì giải thể; quỹ không chia về ví cá nhân
	}
	return g.saveLocked()
}

func (g *GuildStore) AddFund(id string, n int) (*Guild, error) {
	return g.Do(id, func(gd *Guild) error {
		gd.Fund += n
		gd.Members[id].Contributed += n
		gd.ledger(gd.Members[id].Name, n, "Góp quỹ")
		return nil
	})
}

func (g *GuildStore) UndoFund(id string, n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.of(id)
	if gd == nil {
		return
	}
	gd.Fund -= n
	if mb := gd.Members[id]; mb != nil {
		mb.Contributed -= n
	}
	gd.Ledger = gd.Ledger[:max(0, len(gd.Ledger)-1)]
	if err := g.saveLocked(); err != nil {
		log.Printf("guild undo fund %s: %v", id, err) // giữ phần trừ trong bộ nhớ; lần lưu hội sau ghi lại
	}
}

func (g *GuildStore) Touch(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if gd := g.of(id); gd != nil {
		gd.Members[id].LastSeen = time.Now()
	}
}

// AddWin cộng điểm thi đua tuần khi thành viên thắng PK (mục 13: thi đấu hội qua mini-game).
func (g *GuildStore) AddWin(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.of(id)
	if gd == nil {
		return
	}
	if wk := weekKey(time.Now()); gd.Week != wk {
		gd.Week, gd.WeekScore = wk, map[string]int{}
	}
	gd.WeekScore[id]++
	_ = g.saveLocked()
}

type GuildSummary struct {
	ID, Name, Emblem, Color, JoinMode string
	Members, Score                    int
}

func (g *GuildStore) List() []GuildSummary {
	g.mu.Lock()
	defer g.mu.Unlock()
	wk := weekKey(time.Now())
	out := []GuildSummary{}
	for _, gd := range g.Guilds {
		score := 0
		if gd.Week == wk {
			for _, s := range gd.WeekScore {
				score += s
			}
		}
		out = append(out, GuildSummary{gd.ID, gd.Name, gd.Emblem, gd.Color, gd.JoinMode, len(gd.Members), score})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func (g *GuildStore) View(id string) *Guild {
	g.mu.Lock()
	defer g.mu.Unlock()
	gd := g.of(id)
	if gd == nil {
		return nil
	}
	b, _ := json.Marshal(gd)
	var cp Guild
	json.Unmarshal(b, &cp)
	return &cp
}

// ---------- Thao tác cấp phiên ----------

func (h *Hub) guild(s *Session, m *inMsg) {
	me, name := s.acc.ID, s.acc.Char.Name
	gs := h.store.Guilds
	fail := func(err error) { s.Send(map[string]any{"t": "toast", "kind": "warn", "text": err.Error()}) }
	ok := func(text string) {
		if text != "" {
			s.Send(map[string]any{"t": "toast", "kind": "good", "text": text})
		}
		h.sendGuild(s)
	}
	can := func(gd *Guild, roles ...string) error {
		if !contains(roles, gd.role(me)) {
			return errors.New("Vai trò của bạn không có quyền này.")
		}
		return nil
	}
	target := func() (string, error) {
		if validAccountID(m.ID) {
			return m.ID, nil
		}
		return h.resolve(s, m)
	}
	switch m.T {
	case "guild":
		h.sendGuild(s)
	case "guild_list":
		s.Send(map[string]any{"t": "guild_list", "guilds": gs.List()})
	case "guild_join":
		res, err := gs.Join(me, name, m.ID)
		if err != nil {
			fail(err)
			return
		}
		if res == "joined" {
			ok("Đã vào hội!")
			h.guildPush(m.ID, map[string]any{"t": "guild_chat", "system": true, "text": name + " đã vào hội."})
		} else {
			ok("Đã gửi đơn xin vào hội, chờ chủ/phó hội duyệt.")
		}
	case "guild_approve", "guild_deny":
		_, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner", "vice"); err != nil {
				return err
			}
			n, ok := gd.Requests[m.ID]
			if !ok {
				return errors.New("Đơn không còn.")
			}
			if m.T == "guild_deny" {
				delete(gd.Requests, m.ID)
				return nil
			}
			if err := gs.addMemberLocked(gd, m.ID, n); err != nil {
				return err
			}
			gd.ledger(n, 0, "Vào hội")
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		ok("")
	case "guild_invite":
		id, err := target()
		if err != nil {
			fail(err)
			return
		}
		gd, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner", "vice"); err != nil {
				return err
			}
			gd.Invites[id] = time.Now()
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		if o := h.sessionOf(id); o != nil {
			o.Send(map[string]any{"t": "guild_invite", "id": gd.ID, "name": gd.Name, "from": name})
		}
		ok("Đã mời vào hội.")
	case "guild_leave":
		gid, _, _ := gs.GuildOf(me)
		if err := gs.Leave(me); err != nil {
			fail(err)
			return
		}
		h.guildPush(gid, map[string]any{"t": "guild_chat", "system": true, "text": name + " đã rời hội."})
		ok("Đã rời hội.")
	case "guild_kick", "guild_role":
		_, err := gs.Do(me, func(gd *Guild) error {
			t := gd.Members[m.ID]
			if t == nil || m.ID == me {
				return errors.New("Không tìm thấy thành viên.")
			}
			if m.T == "guild_kick" {
				// Phó hội chỉ mời được thành viên thường ra khỏi hội.
				if gd.role(me) == "owner" || (gd.role(me) == "vice" && t.Role == "member") {
					delete(gd.Members, m.ID)
					delete(gs.member, m.ID)
					gd.ledger(t.Name, 0, "Bị mời khỏi hội")
					return nil
				}
				return errors.New("Vai trò của bạn không có quyền này.")
			}
			if err := can(gd, "owner"); err != nil {
				return err
			}
			if m.Mode != "vice" && m.Mode != "member" {
				return errors.New("Vai trò không hợp lệ.")
			}
			t.Role = m.Mode
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		ok("")
	case "guild_transfer":
		// Chuyển quyền chủ hội cần người nhận xác nhận (mục 13).
		_, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner"); err != nil {
				return err
			}
			if gd.Members[m.ID] == nil || m.ID == me {
				return errors.New("Chọn một thành viên khác.")
			}
			gd.Transfer = m.ID
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		if o := h.sessionOf(m.ID); o != nil {
			o.Send(map[string]any{"t": "toast", "kind": "good", "text": name + " đề nghị bạn làm chủ hội. Mở bảng Hội để nhận."})
		}
		ok("Đã gửi đề nghị chuyển quyền, chờ người nhận xác nhận.")
	case "guild_transfer_accept", "guild_claim":
		_, err := gs.Do(me, func(gd *Guild) error {
			if m.T == "guild_transfer_accept" {
				if gd.Transfer != me {
					return errors.New("Không có đề nghị chuyển quyền cho bạn.")
				}
			} else {
				// Chủ hội vắng quá thời hạn công bố thì phó hội nhận quyền, không tự xoá hội.
				if gd.role(me) != "vice" {
					return errors.New("Chỉ phó hội được nhận quyền khi chủ hội vắng lâu.")
				}
				if time.Since(gd.Members[gd.Owner].LastSeen) < h.bal.OwnerAbsent {
					return fmt.Errorf("Chủ hội mới vắng mặt chưa tới %s.", humanDur(h.bal.OwnerAbsent))
				}
			}
			gd.Members[gd.Owner].Role = "vice"
			gd.Owner = me
			gd.Members[me].Role = "owner"
			gd.Transfer = ""
			gd.ledger(name, 0, "Nhận chức chủ hội")
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		ok("Bạn đã là chủ hội.")
	case "guild_mode":
		_, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner", "vice"); err != nil {
				return err
			}
			if m.Mode != "open" && m.Mode != "approval" {
				return errors.New("Chế độ không hợp lệ.")
			}
			gd.JoinMode = m.Mode
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		ok("")
	case "guild_post":
		text := strings.TrimSpace(m.Text)
		if text == "" || len([]rune(text)) > 200 {
			fail(errors.New("Thông báo 1–200 ký tự."))
			return
		}
		_, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner", "vice"); err != nil {
				return err
			}
			gd.Board = append(gd.Board, GuildPost{name, text, time.Now()})
			if len(gd.Board) > 30 {
				gd.Board = gd.Board[len(gd.Board)-30:]
			}
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		ok("Đã đăng lên bảng hội.")
	case "guild_project":
		var p *GuildProject
		for i := range GuildProjects {
			if GuildProjects[i].ID == m.ID {
				p = &GuildProjects[i]
			}
		}
		gd, err := gs.Do(me, func(gd *Guild) error {
			if err := can(gd, "owner", "vice"); err != nil {
				return err
			}
			switch {
			case p == nil:
				return errors.New("Công trình không hợp lệ.")
			case gd.Projects[p.ID]:
				return errors.New("Công trình này đã xây.")
			case gd.Fund < p.Cost:
				return fmt.Errorf("Quỹ hội cần %d thóc.", p.Cost)
			}
			gd.Fund -= p.Cost
			gd.Projects[p.ID] = true
			gd.ledger(name, -p.Cost, "Xây "+p.Name)
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		h.guildPush(gd.ID, map[string]any{"t": "guild_chat", "system": true, "text": "🏗️ Hội đã hoàn thành " + p.Name + "!"})
		ok("")
	case "guild_hall":
		if !h.inResidential(s) {
			fail(errors.New("Nhà Hội nằm ở Xóm Rơm Vàng. Mở bản đồ để đặt chỉ đường."))
			return
		}
		if err := h.VisitGuild(s); err != nil {
			fail(err)
		}
	case "guild_chat":
		if until, reason, muted := h.store.Mod.Muted(me); muted {
			fail(fmt.Errorf("Bạn đang bị khoá chat đến %s (%s).", until.In(vnZone).Format("15:04"), reason))
			return
		}
		text := strings.TrimSpace(m.Text)
		gid, _, _ := gs.GuildOf(me)
		if gid == "" || text == "" {
			return
		}
		if len([]rune(text)) > ChatMaxLen {
			text = string([]rune(text)[:ChatMaxLen])
		}
		h.guildPush(gid, map[string]any{"t": "guild_chat", "name": name, "text": text, "at": time.Now()})
	}
}

// Nhà Hội (ảnh "Sân họp gà"): sân họp riêng của hội, chỉ thành viên vào được.
func guildHallZone(name string) *Zone {
	return &Zone{ID: "guildhall", Title: "Nhà Hội " + name, HalfX: 10.4, HalfZ: 8.6,
		Spawns: []Vec{{0, 6.8}, {-1.2, 6.8}, {1.2, 6.8}, {0, 5.6}, {-2, 5.8}, {2, 5.8}},
		Items: []Item{
			{ID: "ve_sanh", Kind: "home_exit", Label: "Ra Xóm Rơm Vàng", Emo: "🏠", X: 0, Z: 8.0, R: 1.4},
			{ID: "bang_hoi", Kind: "guild_board", Label: "Bảng hội", Emo: "📜", X: 7.2, Z: -4.6, R: 2.4},
		}}
}

func NewGuildRoom(h *Hub, gid, name string) *Room {
	r := NewRoom(h, "g:"+gid, "Nhà Hội "+name)
	r.Kind, r.Owner, r.entry, r.cap = "guild", "g:"+gid, "guildhall", GuildMax
	r.zones = map[string]*Zone{"guildhall": guildHallZone(name)}
	return r
}

// VisitGuild đưa thành viên vào Nhà Hội của hội mình.
func (h *Hub) VisitGuild(s *Session) error {
	s.moveMu.Lock()
	defer s.moveMu.Unlock()
	if s.kicked.Load() || s.room.Load() == nil {
		return errors.New("Phiên này không còn trong game.")
	}
	gid, name, _ := h.store.Guilds.GuildOf(s.acc.ID)
	if gid == "" {
		return errors.New("Bạn chưa ở trong hội nào.")
	}
	cur := s.room.Load()
	if cur != nil && cur.Owner == "g:"+gid {
		return nil
	}
	if cur != nil {
		ok := true
		cur.DoWait(func() { ok = cur.canLeave(s) })
		if !ok {
			return errors.New("Đang trong trận hoặc trên xe, chưa đi được.")
		}
	}
	h.mu.Lock()
	target := h.homes["g:"+gid]
	if target == nil {
		target = NewGuildRoom(h, gid, name)
		h.homes["g:"+gid] = target
		go target.Run()
	}
	if target.Count() >= GuildMax {
		h.mu.Unlock()
		return errors.New("Nhà Hội đang đủ người.")
	}
	target.reserve()
	h.mu.Unlock()
	if cur != nil {
		cur.DoWait(func() { cur.remove(s) })
	}
	s.room.Store(target)
	target.DoWait(func() { target.add(s) })
	return nil
}

// guildPush gửi tin tới mọi thành viên đang online, ở bất kỳ phòng nào (chat hội xuyên phòng).
func (h *Hub) guildPush(gid string, v map[string]any) {
	for _, id := range h.store.Guilds.Members(gid) {
		if s := h.sessionOf(id); s != nil {
			s.Send(v)
		}
	}
}

func (h *Hub) sendGuild(s *Session) {
	h.store.Guilds.Touch(s.acc.ID)
	s.Send(map[string]any{"t": "guild", "guild": h.store.Guilds.View(s.acc.ID), "projects": GuildProjects,
		"emblems": GuildEmblems, "colors": GuildColors, "me": s.acc.ID})
}

// guildAct: phần chạm ví chạy trong goroutine phòng (lập hội, góp quỹ).
func (r *Room) guildAct(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn txnFunc) {
	now := time.Now()
	gs := r.hub.store.Guilds
	switch m.A {
	case "guild_create":
		if p.acc.Level < GuildLevel {
			fail(fmt.Errorf("Lập hội cần cấp %d.", GuildLevel))
			return
		}
		if gid, _, _ := gs.GuildOf(p.acc.ID); gid != "" {
			fail(errors.New("Bạn đang ở trong một hội."))
			return
		}
		created := false
		ok := txn(func() (ActResult, error) {
			if err := p.acc.addThoc(-GuildCost, "guild_create", m.Text, now); err != nil {
				return ActResult{}, err
			}
			gd, err := gs.Create(p.acc.ID, p.acc.Char.Name, m.Text, m.ID, m.Mode)
			if err != nil {
				return ActResult{}, err
			}
			created = true
			r.broadcastLook(p)
			return ActResult{Msg: "Đã lập " + gd.Emblem + " " + gd.Name + "!"}, nil
		})
		if !ok && created {
			// Phí lập hội đã hoàn nguyên: giải thể hội vừa lập (một thành viên rời là hội tự giải thể), không có hội miễn phí (M4).
			if err := gs.Leave(p.acc.ID); err != nil {
				log.Printf("guild create rollback %s: %v", p.acc.ID, err)
			}
			r.broadcastLook(p)
		}
	case "guild_give":
		n := m.I
		if n <= 0 || n > p.acc.Thoc {
			fail(ErrNotEnough)
			return
		}
		gd, err := gs.AddFund(p.acc.ID, n)
		if err != nil {
			fail(err)
			return
		}
		snap, _ := json.Marshal(p.acc)
		_ = p.acc.addThoc(-n, "guild_fund", gd.ID, now)
		res := ActResult{OK: true, Msg: fmt.Sprintf("Đã góp %d thóc vào quỹ %s.", n, gd.Name)}
		p.acc.Remember(m.RID, res)
		if err := r.hub.store.Save(p.acc); err != nil {
			restore(p.acc, snap)
			gs.UndoFund(p.acc.ID, n)
			fail(errors.New("Máy chủ tạm không lưu được, chưa trừ thóc."))
			return
		}
		reply(res)
	}
}
