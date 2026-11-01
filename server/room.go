package main

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// Room là một chuồng công cộng: sảnh + vườn giun + chợ + sân tỷ thí, tối đa 25 người.
// Mọi trạng thái của phòng chỉ được đọc/ghi trong goroutine Run (qua Do/DoWait).

type Player struct {
	PID  string
	sess *Session
	acc  *Account

	zone   string
	pos    Vec
	ry     float64
	in     Vec
	inAt   time.Time
	moving bool

	dig    *DigSession
	match  *Match
	invite *Invite

	chatTokens  float64
	chatAt      time.Time
	emoteAt     time.Time
	lastStage   int
	vitAt       time.Time
	dirty       bool
	lastInvited map[string]time.Time
	goneAt      time.Time            // mất kết nối, đang giữ chỗ chờ nối lại
	ride        *Ride                // xe mình đang lái
	rideOf      *Player              // người lái của xe mình đang ngồi
	seatInvites map[string]time.Time // pid người lái → hạn lời mời lên xe
	trade       *Trade
}

type DigSession struct {
	ID     string
	Spot   string
	Start  time.Time
	Center float64
	Half   float64
	Speed  float64
	MinDur time.Duration
}

// ChatLog là bản ghi chat phía máy chủ, dùng làm bằng chứng khi báo cáo (mục 11).
type ChatLog struct {
	ID   string    `json:"id"`
	From string    `json:"from"`
	Name string    `json:"name"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type Room struct {
	ID, Name string
	Kind     string // "public" | "home"
	Region   string // vùng của chuồng công cộng
	buses    map[string]*Bus
	Owner    string // chủ nhà với phòng "home"
	home     *Home
	zones    map[string]*Zone
	entry    string
	cap      int
	quit     bool
	chatLog  []ChatLog
	hub      *Hub
	inbox    chan func()
	players  map[*Session]*Player
	count    atomic.Int32
	rng      *rand.Rand
	tick     uint64
	matches  map[string]*Match
	invites  map[string]*Invite
	pidSeq   int

	tickMu  sync.Mutex
	tickBuf []float64 // thời gian xử lý tick gần nhất (ms) cho số đo vận hành
}

func (r *Room) TickSamples() []float64 {
	r.tickMu.Lock()
	defer r.tickMu.Unlock()
	return append([]float64(nil), r.tickBuf...)
}

func NewRoom(h *Hub, id, name string) *Room {
	var seed [32]byte
	_, _ = crand.Read(seed[:])
	return &Room{ID: id, Name: name, Kind: "public", Region: RegionHypex, zones: Zones, entry: ZoneLobby, cap: RoomCapacity, buses: map[string]*Bus{},
		hub: h, inbox: make(chan func(), 1024), players: map[*Session]*Player{},
		rng: rand.New(rand.NewChaCha8(seed)), matches: map[string]*Match{}, invites: map[string]*Invite{}}
}

// NewRegionRoom mở một bản sao chuồng công cộng của vùng (trại Hypex hoặc trại Ven Suối).
func NewRegionRoom(h *Hub, id, name, region string) *Room {
	r := NewRoom(h, id, name)
	reg := Regions[region]
	r.Region, r.zones, r.entry = region, reg.Zones, reg.Entry
	return r
}

// NewHomeRoom mở phòng riêng cho nhà của một người (tối đa 8 người, mục 9.1).
func NewHomeRoom(h *Hub, home *Home) *Room {
	r := NewRoom(h, "h:"+home.Owner, "Chuồng riêng của "+home.OwnerName)
	r.Kind, r.Owner, r.home, r.entry, r.cap = "home", home.Owner, home, "home", HomeCapacity
	r.zones = map[string]*Zone{"home": home.Zone()}
	return r
}

func (r *Room) Count() int  { return int(r.count.Load()) }
func (r *Room) reserve()    { r.count.Add(1) }
func (r *Room) unreserve()  { r.count.Add(-1) }
func (r *Room) Do(f func()) { r.inbox <- f }

func (r *Room) DoWait(f func()) {
	done := make(chan struct{})
	r.inbox <- func() { f(); close(done) }
	<-done
}

func (r *Room) Run() {
	t := time.NewTicker(time.Second / TickRate)
	defer t.Stop()
	last := time.Now()
	for !r.quit {
		select {
		case f := <-r.inbox:
			f()
		case now := <-t.C:
			dt := now.Sub(last).Seconds()
			last = now
			r.step(now, dt)
			ms := float64(time.Since(now).Microseconds()) / 1000
			r.tickMu.Lock()
			r.tickBuf = append(r.tickBuf, ms)
			if len(r.tickBuf) > 600 {
				r.tickBuf = r.tickBuf[len(r.tickBuf)-600:]
			}
			r.tickMu.Unlock()
		}
	}
}

// ---------- Vào / ra phòng ----------

func (r *Room) add(s *Session) {
	r.pidSeq++
	p := &Player{PID: "p" + strconv.Itoa(r.pidSeq), sess: s, acc: s.acc, zone: r.entry, lastInvited: map[string]time.Time{}, seatInvites: map[string]time.Time{}}
	arr := s.arrive
	s.arrive = nil
	r.arriveV11(p, arr, time.Now())
	p.pos = r.freeSpawn(r.zones[p.zone])
	p.ry = math.Pi
	p.lastStage = p.acc.HungerStage(r.hub.bal)
	r.players[s] = p
	now := time.Now()

	r.discover(p, "zone:"+p.zone)
	r.sendWelcome(p, now)
	r.broadcastZone(p.zone, p, map[string]any{"t": "pjoin", "p": r.publicOf(p)})
	r.broadcastRoom(nil, map[string]any{"t": "room", "room": r.info()})
	r.system(p.acc.Char.Name + " vừa vào chuồng.")
	if r.home != nil && p.acc.ID != r.Owner {
		// Thư Chim Chíp báo chủ nhà có khách (mục 11).
		owner, name := r.Owner, p.acc.Char.Name
		go func() {
			if r.hub.store.Social.AddMail(owner, "visit", "🐦 "+name+" vừa ghé thăm chuồng của bạn.") {
				r.hub.mailPing(owner)
			}
		}()
	}
}

func (r *Room) sendWelcome(p *Player, now time.Time) {
	welcome := map[string]any{
		"t": "welcome", "pid": p.PID, "me": p.acc.View(now, r.hub.bal),
		"room": r.info(), "zones": r.zones, "zone": p.zone, "pos": p.pos, "ry": p.ry,
		"players": r.zonePlayers(p.zone, p), "catalog": catalog(r.hub.bal), "festival": r.hub.festival.Snapshot(),
		"self": r.publicOf(p), "bus": r.busInfo(), "region": r.Region,
	}
	if r.home != nil {
		r.home.Tick(now)
		welcome["home"] = r.home.View(now)
	}
	if r.Kind == "guild" {
		if g := r.hub.store.Guilds.View(p.acc.ID); g != nil {
			welcome["guildhall"] = map[string]any{"name": g.Name, "emblem": g.Emblem, "color": g.Color, "projects": g.Projects}
		}
	}
	p.sess.Send(welcome)
}

// markGone giữ con gà trong phòng khi mất kết nối; hết ReconnectGrace thì mới rời phòng.
func (r *Room) markGone(s *Session) {
	p := r.players[s]
	if p == nil {
		go r.hub.dropSession(s)
		return
	}
	p.goneAt = time.Now()
	p.in = Vec{}
	p.dig = nil
	r.broadcastZone(p.zone, p, map[string]any{"t": "pgone", "pid": p.PID, "gone": true})
}

// rebind gắn phiên mới vào con gà đang giữ chỗ. Đang trong trận PK thì luôn nối lại (kể cả tải lại trang);
// ngoài trận chỉ nối lại khi client tự kết nối lại, còn mở trang mới là phiên mới về sảnh (mục 6.3).
func (r *Room) rebind(old, s *Session, resume bool) bool {
	p := r.players[old]
	if p == nil || p.goneAt.IsZero() || (!resume && p.match == nil) {
		return false
	}
	delete(r.players, old)
	r.players[s] = p
	p.sess = s
	p.goneAt = time.Time{}
	now := time.Now()
	r.sendWelcome(p, now)
	r.broadcastZone(p.zone, p, map[string]any{"t": "pgone", "pid": p.PID, "gone": false})
	if mt := p.match; mt != nil {
		s.Send(r.pkResume(mt, p, now))
	}
	return true
}

func (r *Room) discover(p *Player, key string) {
	for _, n := range p.acc.Discovered(key) {
		r.toast(p, "good", n)
		r.broadcastLook(p)
	}
}

func (r *Room) canLeave(s *Session) bool {
	p := r.players[s]
	return p == nil || (p.match == nil && p.ride == nil && p.rideOf == nil)
}

func (r *Room) remove(s *Session) {
	p := r.players[s]
	if p == nil {
		return
	}
	if p.trade != nil {
		r.closeTrade(p.trade, "Bên kia đã rời đi.")
	}
	r.parkVehicle(p, "Người lái đã rời đi, bạn xuống xe an toàn.")
	if p.rideOf != nil {
		r.dismount(p, "")
	}
	r.cancelTrip(p, "Bạn đã rời bến trước giờ xe chạy.")
	r.endMatchFor(p, "left")
	r.cancelInvitesFor(p)
	delete(r.players, s)
	r.count.Add(-1)
	if err := r.hub.store.Save(p.acc); err != nil {
		log.Printf("save on leave %s: %v", p.acc.ID, err)
	}
	r.broadcastZone(p.zone, nil, map[string]any{"t": "pleave", "pid": p.PID})
	r.broadcastRoom(nil, map[string]any{"t": "room", "room": r.info()})
	r.system(p.acc.Char.Name + " đã rời chuồng.")
	if p.sess.closed.Load() && !p.sess.kicked.Load() {
		// Gỡ liên kết phòng trước: phòng nhà có thể đóng ngay sau đây, dropSession không được chờ nó.
		p.sess.room.CompareAndSwap(r, nil)
		go r.hub.dropSession(p.sess)
	}
	if r.Kind != "public" && len(r.players) == 0 {
		go r.hub.closeHome(r)
	}
}

// Chọn điểm xuất hiện xa người khác nhất để không chồng lên nhau (mục 6.4).
func (r *Room) freeSpawn(z *Zone) Vec {
	best, bestD := z.Spawns[0], -1.0
	for _, sp := range z.Spawns {
		d := 99.0
		for _, o := range r.players {
			if o.zone == z.ID {
				d = math.Min(d, sp.Dist(o.pos))
			}
		}
		d += r.rng.Float64() * 0.3
		if d > bestD {
			best, bestD = sp, d
		}
	}
	return best
}

func (r *Room) info() map[string]any {
	return map[string]any{"id": r.ID, "name": r.Name, "count": r.Count(), "cap": r.cap, "kind": r.Kind, "owner": r.Owner}
}

type publicPlayer struct {
	PID      string            `json:"pid"`
	Name     string            `json:"name"`
	Tag      string            `json:"tag"`
	Char     *Character        `json:"char"`
	Equipped map[string]string `json:"equipped"`
	Level    int               `json:"level"`
	Title    string            `json:"title"`
	Vehicle  string            `json:"vehicle,omitempty"`
	Seats    []string          `json:"seats,omitempty"`
	RideOf   string            `json:"rideOf,omitempty"`
	Pet      *petView          `json:"pet,omitempty"`
	Guild    string            `json:"guild,omitempty"`
	GColor   string            `json:"gcolor,omitempty"`
	Gone     bool              `json:"gone"`
	X        float64           `json:"x"`
	Z        float64           `json:"z"`
	RY       float64           `json:"ry"`
	Stage    int               `json:"stage"`
}

// Chỉ gửi thông tin công khai; ví/túi đầy đủ không broadcast (mục 16.3).
func (r *Room) publicOf(p *Player) publicPlayer {
	eq := map[string]string{}
	for k, v := range p.acc.Equipped {
		eq[k] = v
	}
	pp := publicPlayer{PID: p.PID, Name: p.acc.Char.Name, Tag: PlayerTag(p.acc.ID), Char: p.acc.Char, Equipped: eq, Level: p.acc.Level,
		Title: titleName(p.acc.Title), Gone: !p.goneAt.IsZero(), X: p.pos.X, Z: p.pos.Z, RY: p.ry, Stage: p.acc.HungerStage(r.hub.bal), Pet: petOf(p.acc)}
	if p.ride != nil {
		pp.Vehicle = p.ride.Kind
		for _, q := range p.ride.Seats {
			if q == nil {
				pp.Seats = append(pp.Seats, "")
			} else {
				pp.Seats = append(pp.Seats, q.PID)
			}
		}
	}
	if p.rideOf != nil {
		pp.RideOf = p.rideOf.PID
	}
	_, pp.Guild, pp.GColor = r.hub.store.Guilds.GuildOf(p.acc.ID)
	return pp
}

func (r *Room) zonePlayers(zone string, except *Player) []publicPlayer {
	out := []publicPlayer{}
	for _, o := range r.players {
		if o != except && o.zone == zone {
			out = append(out, r.publicOf(o))
		}
	}
	return out
}

func (r *Room) byPID(pid string) *Player {
	for _, p := range r.players {
		if p.PID == pid {
			return p
		}
	}
	return nil
}

func (r *Room) broadcastZone(zone string, except *Player, v any) {
	b, _ := json.Marshal(v)
	for _, p := range r.players {
		if p != except && p.zone == zone {
			p.sess.sendRaw(b)
		}
	}
}

func (r *Room) broadcastRoom(except *Player, v any) {
	b, _ := json.Marshal(v)
	for _, p := range r.players {
		if p != except {
			p.sess.sendRaw(b)
		}
	}
}

// sendRaw gửi không chặn. Kiểm tra "đã đóng" và gửi nằm chung khoá với Close
// để không bao giờ gửi vào channel vừa bị đóng bởi goroutine khác.
func (s *Session) sendRaw(b []byte) {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.closed.Load() {
		return
	}
	select {
	case s.send <- b:
	default:
		// Client quá chậm: ngắt để không giữ bộ nhớ phòng; client sẽ kết nối lại.
		sendDrops.Add(1)
		s.closeLocked()
	}
}

func (r *Room) system(text string) {
	r.broadcastRoom(nil, map[string]any{"t": "chat", "kind": "system", "text": text})
}

// ---------- Mô phỏng ----------

func (r *Room) step(now time.Time, dt float64) {
	r.tick++
	b := r.hub.bal
	for s, p := range r.players {
		if !p.goneAt.IsZero() {
			// Hết giờ giữ chỗ: rời phòng như thoát game (trận PK đang đấu sẽ xử thua).
			if now.Sub(p.goneAt) > b.ReconnectGrace {
				r.remove(s)
			}
			continue
		}
		p.acc.TickVitals(dt, b)
		stage := p.acc.HungerStage(b)
		if stage != p.lastStage || now.Sub(p.vitAt) > time.Second {
			if stage != p.lastStage {
				r.broadcastZone(p.zone, nil, map[string]any{"t": "pstage", "pid": p.PID, "stage": stage})
				if stage == 4 && p.dig != nil {
					p.dig = nil
				}
			}
			p.lastStage = stage
			p.vitAt = now
			p.dirty = true
			p.sess.Send(map[string]any{"t": "vit", "mana": math.Round(p.acc.Mana*10) / 10, "stage": stage, "zero": p.acc.ZeroElapsed})
		}

		// Di chuyển do máy chủ tính từ hướng input; input cũ hơn 300ms coi như dừng.
		mult := 1.0
		if p.ride != nil {
			mult = vehicleByID(p.ride.Kind).Speed
			if stage >= 4 {
				mult = 0 // kiệt sức khi đang lái: xe giảm tốc và dừng an toàn (mục 7.4)
			}
		}
		if p.rideOf != nil {
			continue // khách ngồi theo ghế, vị trí do syncRides đặt
		}
		if now.Sub(p.inAt) < 300*time.Millisecond && (p.in.X != 0 || p.in.Z != 0) && p.dig == nil && mult > 0 {
			spd := WalkSpeed * SpeedFactor(stage) * mult * dt
			z := r.zones[p.zone]
			p.pos = z.Clamp(Vec{p.pos.X + p.in.X*spd, p.pos.Z + p.in.Z*spd})
			p.ry = math.Atan2(p.in.X, p.in.Z)
			p.moving = true
		} else {
			p.moving = false
		}
	}
	r.stepPK(now)
	r.stepV11(now)
	if r.home != nil && r.home.Tick(now) {
		if err := r.hub.store.Homes.Save(r.home); err != nil {
			log.Printf("save home %s: %v", r.Owner, err)
		}
		r.broadcastHome()
		r.system("🐮 Bò Mộc: chuồng đã xây xong!")
	}

	if r.tick%SnapEvery == 0 {
		byZone := map[string][][]float64{}
		pidsByZone := map[string][]string{}
		for _, p := range r.players {
			mv := 0.0
			if p.moving {
				mv = 1
			}
			byZone[p.zone] = append(byZone[p.zone], []float64{round2(p.pos.X), round2(p.pos.Z), round2(p.ry), mv})
			pidsByZone[p.zone] = append(pidsByZone[p.zone], p.PID)
		}
		for zone, rows := range byZone {
			r.broadcastZone(zone, nil, map[string]any{"t": "snap", "tick": r.tick, "pids": pidsByZone[zone], "p": rows})
		}
	}
	// Checkpoint chỉ số sống định kỳ; tài sản đã lưu ngay lúc giao dịch.
	if r.tick%(TickRate*10) == 0 {
		for _, p := range r.players {
			if p.dirty {
				if err := r.hub.store.Save(p.acc); err != nil {
					log.Printf("checkpoint %s: %v", p.acc.ID, err)
				}
				p.dirty = false
			}
		}
	}
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// ---------- Xử lý tin nhắn ----------

func (r *Room) handle(s *Session, m *inMsg) {
	p := r.players[s]
	if p == nil {
		return
	}
	switch m.T {
	case "in":
		l := math.Hypot(m.X, m.Z)
		if l > 1 {
			m.X, m.Z = m.X/l, m.Z/l
		}
		if math.IsNaN(l) {
			return
		}
		p.in = Vec{m.X, m.Z}
		p.inAt = time.Now()
	case "chat":
		r.chat(p, m.Text)
	case "emote":
		r.emote(p, m.E)
	case "act":
		r.act(p, m)
	}
}

func (r *Room) chat(p *Player, text string) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return
	}
	if until, reason, muted := r.hub.store.Mod.Muted(p.acc.ID); muted {
		r.toast(p, "bad", fmt.Sprintf("Bạn đang bị khoá chat đến %s (%s).", until.In(vnZone).Format("15:04 02/01"), reason))
		return
	}
	if utf8.RuneCountInString(text) > ChatMaxLen {
		text = string([]rune(text)[:ChatMaxLen])
	}
	// Token bucket: tối đa 3 tin liên tiếp, hồi 1 tin/1,5 giây.
	now := time.Now()
	p.chatTokens = math.Min(3, p.chatTokens+now.Sub(p.chatAt).Seconds()/1.5)
	p.chatAt = now
	if p.chatTokens < 1 {
		p.sess.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Bạn gửi nhanh quá, chờ chút nhé."})
		return
	}
	p.chatTokens--
	p.acc.step("first_social")
	r.pidSeq++
	line := ChatLog{ID: "c" + strconv.Itoa(r.pidSeq), From: p.acc.ID, Name: p.acc.Char.Name, Text: text, At: now}
	r.chatLog = append(r.chatLog, line)
	if len(r.chatLog) > 100 {
		r.chatLog = r.chatLog[len(r.chatLog)-100:]
	}
	b, _ := json.Marshal(map[string]any{"t": "chat", "kind": "chat", "id": line.ID, "pid": p.PID, "name": p.acc.Char.Name, "tag": PlayerTag(p.acc.ID), "zone": p.zone, "text": text})
	for _, o := range r.players {
		// Người đã chặn nhau không nhận chat của nhau (mục 11).
		if o == p || !r.hub.store.Social.Blocked(o.acc.ID, p.acc.ID) {
			o.sess.sendRaw(b)
		}
	}
}

// Evidence lấy các dòng chat gần nhất của một tài khoản từ nhật ký máy chủ.
func (r *Room) Evidence(accID string) []ChatLog {
	var out []ChatLog
	for _, l := range r.chatLog {
		if l.From == accID {
			out = append(out, l)
		}
	}
	if len(out) > 10 {
		out = out[len(out)-10:]
	}
	return out
}

func (r *Room) broadcastHome() {
	r.broadcastRoom(nil, map[string]any{"t": "home", "home": r.home.View(time.Now()), "zone": r.zones["home"]})
}

// Biểu cảm đầu (mục 5).
var Emotes = []string{"chao", "vui", "buon", "ngac_nhien", "mua", "gay", "an_mung", "ngoi", "ngu"}

func (r *Room) emote(p *Player, e string) {
	if !contains(Emotes, e) || time.Since(p.emoteAt) < 800*time.Millisecond {
		return
	}
	p.emoteAt = time.Now()
	if e == "chao" || e == "gay" {
		p.acc.step("first_social")
	}
	r.broadcastZone(p.zone, nil, map[string]any{"t": "emote", "pid": p.PID, "e": e})
}

func (r *Room) toast(p *Player, kind, text string) {
	p.sess.Send(map[string]any{"t": "toast", "kind": kind, "text": text})
}

type petView struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Affinity int    `json:"level"`
}

func petOf(a *Account) *petView {
	c := a.companion(a.Following)
	if c == nil {
		return nil
	}
	lv, _ := AffinityLevel(c.Affinity)
	return &petView{c.Name, c.Color, lv}
}
