package main

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"runtime/debug"
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

	seat   string
	zone   string
	pos    Vec
	ry     float64
	in     Vec
	inAt   time.Time
	moving bool

	dig       *DigSession
	fish      *DigSession // đang câu cá ở ao
	fbedsSent string      // danh sách cây của bạn bè đã gửi (tránh gửi lặp)
	showSent  string      // lô trưng bày đã gửi
	match     *Match
	invite    *Invite

	chatTokens   float64
	chatAt       time.Time
	emoteAt      time.Time
	lastStage    int
	vitAt        time.Time
	dirty        bool
	lastInvited  map[string]time.Time
	crowdToastAt time.Time            // lần cuối báo "khu đang đông" (M9)
	crowdSentAt  time.Time            // lần cuối gửi area_full cho client (chặn dự đoán bước vào khu đầy)
	leaving      bool                 // đang rời phòng (Room.remove)
	pkEnd        map[string]any       // kết quả trận chốt lúc đang mất mạng, gửi lại khi nối lại (M10)
	goneAt       time.Time            // mất kết nối, đang giữ chỗ chờ nối lại
	ride         *Ride                // xe mình đang lái
	rideOf       *Player              // người lái của xe mình đang ngồi
	seatInvites  map[string]time.Time // pid người lái → hạn lời mời lên xe
	trade        *Trade

	// Chạy nhanh (PLAN_LOGIC_GAME mục 3): thể lực tính bằng giây chạy, chỉ sống trong phiên.
	run        bool      // input đang giữ nút chạy
	running    bool      // tick vừa rồi thật sự chạy
	stam       float64   // giây chạy còn lại
	stamInit   bool      // đã nạp đầy lần đầu
	stamLock   time.Time // hụt hơi: khoá tới mốc này rồi đầy lại
	stamUsedAt time.Time
	stamSent   float64
	stamSentAt time.Time

	// Làng sống (PLAN_LOGIC_GAME 5–9): đang ngồi xe NPC (xe buýt / xe ôm), trí nhớ của đạo diễn.
	carrier     *Actor
	carrierSeat int
	busDest     string // mã khu của bến muốn xuống
	alightNext  bool   // xuống ở bến kế
	joinedAt    time.Time
	memo        npcMemo
	gooseUntil  time.Time // cưỡi ngỗng (Ngỗng con Lv3): chạy ×1,5 tới mốc này
	gooseReady  time.Time // hồi kỹ năng cưỡi ngỗng
}

// DigSession là một lượt đào đang mở: loài giun và seed giằng co do máy chủ gieo lúc bắt đầu.
type DigSession struct {
	ID        string
	Spot      string
	Start     time.Time
	TugAt     time.Time // giằng co bắt đầu sau lúc mổ đất và lúc giun ló đầu
	Worm      *Worm
	Seed      uint32
	Params    TugParams
	Exhausted bool
	Fish      *FishDef // lượt câu cá (cùng khung giằng co)
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
	areaQ    map[string][]*areaWait // hàng chờ ở cổng các khu đã đủ người (area_capacity_queue.go)
	world    *Region                // bản đồ liền của chuồng công cộng; nil với nhà riêng / Nhà Hội
	entry    string
	cap      int
	quit     bool
	stopped  chan struct{} // đóng khi Run thoát, để DoWait không chờ phòng đã dừng
	chatLog  []ChatLog
	hub      *Hub
	inbox    chan func()
	players  map[*Session]*Player
	count    atomic.Int32
	rng      *rand.Rand
	tick     uint64
	matches  map[string]*Match
	invites  map[string]*Invite
	pkb      pkBoard // bảng điểm 3D ở Sân tỷ thí (pk_scoreboard.go)
	pidSeq   int

	actors      map[string]*Actor // NPC di động: xe buýt, xe ôm, xe trên đường, cameo
	actorSeq    int
	actorsReady bool
	show        *arenaShow // trận biểu diễn đang diễn ở Sân tỷ thí
	showNext    time.Time
	clockSent   *Clock    // ngày/đêm, thời tiết đã báo cho phòng
	fbedsDirty  bool      // có người vào/ra hoặc cây đổi: gửi lại cây của bạn bè
	sayWin      time.Time // hạn mức toàn phòng: tối đa SayPerSec câu NPC không nhắm ai mỗi giây
	sayN        int

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
	return &Room{ID: id, Name: name, Kind: "public", Region: RegionHypex, zones: Zones, world: Regions[RegionHypex], entry: ZoneLobby, cap: RoomCapacity, buses: map[string]*Bus{},
		hub: h, inbox: make(chan func(), 1024), stopped: make(chan struct{}), players: map[*Session]*Player{},
		rng: rand.New(rand.NewChaCha8(seed)), matches: map[string]*Match{}, invites: map[string]*Invite{}}
}

// NewRegionRoom mở một bản sao chuồng công cộng của vùng (trại Hypex hoặc trại Ven Suối).
func NewRegionRoom(h *Hub, id, name, region string) *Room {
	r := NewRoom(h, id, name)
	reg := Regions[region]
	r.Region, r.zones, r.entry, r.world = region, reg.Zones, reg.Entry, reg
	return r
}

// NewHomeRoom mở phòng riêng cho nhà của một người (tối đa 8 người, mục 9.1).
func NewHomeRoom(h *Hub, home *Home) *Room {
	r := NewRoom(h, "h:"+home.Owner, "Chuồng riêng của "+home.OwnerName)
	_, cap := home.comfort()
	r.Kind, r.Owner, r.home, r.entry, r.cap = "home", home.Owner, home, "home", cap
	r.zones = map[string]*Zone{"home": home.Zone()}
	r.world = nil
	return r
}

func (r *Room) Count() int  { return int(r.count.Load()) }
func (r *Room) reserve()    { r.count.Add(1) }
func (r *Room) unreserve()  { r.count.Add(-1) }
func (r *Room) Do(f func()) { r.inbox <- f }

// DoWait chạy f trên goroutine phòng và chờ xong. Phòng đã dừng (nhà/Nhà Hội đóng khi hết người) thì trả về
// ngay thay vì chờ mãi (lỗi thấp L9); khi đó f có thể không chạy.
func (r *Room) DoWait(f func()) {
	done := make(chan struct{})
	select {
	case r.inbox <- func() { defer close(done); f() }: // defer: f panic (đã được safely bắt) vẫn nhả người chờ
	case <-r.stopped:
		return
	}
	select {
	case <-done:
	case <-r.stopped:
	}
}

// logPanic ghi panic kèm stack để điều tra; không có nó một lỗi nil trong handler sẽ giết cả tiến trình.
func logPanic(where string, e any) {
	log.Printf("PANIC %s: %v\n%s", where, e, debug.Stack())
}

// safely chạy f trên goroutine phòng; panic chỉ huỷ việc đang xử lý, phòng và các phòng khác vẫn chạy.
func (r *Room) safely(what string, f func()) {
	defer func() {
		if e := recover(); e != nil {
			logPanic("room "+r.ID+" "+what, e)
		}
	}()
	f()
}

func (r *Room) Run() {
	t := time.NewTicker(time.Second / TickRate)
	defer t.Stop()
	defer close(r.stopped)
	last := time.Now()
	for !r.quit {
		select {
		case f := <-r.inbox:
			r.safely("inbox", f)
		case now := <-t.C:
			dt := now.Sub(last).Seconds()
			last = now
			r.safely("tick", func() { r.step(now, dt) })
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
	r.fbedsDirty = true
	r.pidSeq++
	p := &Player{PID: "p" + strconv.Itoa(r.pidSeq), sess: s, acc: s.acc, zone: r.entry, lastInvited: map[string]time.Time{}, seatInvites: map[string]time.Time{},
		joinedAt: time.Now(), memo: npcMemo{greetAt: map[string]time.Time{}}}
	arr := s.arrive
	s.arrive = nil
	r.arriveV11(p, arr, time.Now())
	p.pos = r.freeSpawn(r.zones[p.zone])
	if arr != nil && arr.Pos != nil && r.world != nil && r.world.World.Walkable(*arr.Pos) {
		p.pos = *arr.Pos
		p.zone = r.world.AreaAt(p.pos)
	}
	p.ry = math.Pi
	p.lastStage = p.acc.HungerStage(r.hub.bal)
	r.players[s] = p
	crowded := r.keepOutOfFullArea(p) // đổi kênh/tới vùng ngay trong khu đã đầy: đứng ngoài cổng
	if crowded != "" {
		p.zone = r.world.AreaAt(p.pos)
	}
	now := time.Now()
	if s.closed.Load() {
		// Socket đóng ngay trước khi vào phòng: Leave không thấy phòng này nên không đánh dấu. Tự đánh dấu để
		// hết giờ giữ chỗ thì gỡ ra, không để lại gà "ma" vĩnh viễn.
		p.goneAt = now
	}
	if r.home != nil && p.acc.ID == r.Owner && r.home.houseDone() && !p.acc.HouseBuilt {
		p.acc.HouseBuilt, p.dirty = true, true // chuồng nuôi thêm 6 bạn đồng hành
	}
	// Quà khởi nghiệp 100 thóc: tài khoản mới và tài khoản cũ đều nhận đúng một lần (PLAN_LOGIC_GAME 2.3).
	starter := p.acc.GrantStarter(now, r.hub.bal)
	if starter {
		if err := r.hub.store.Save(p.acc); err != nil {
			log.Printf("starter %s: %v", p.acc.ID, err)
		}
	}
	defer func() { // gửi sau welcome để client đã sẵn sàng hiện thông báo
		if crowded != "" {
			r.toast(p, "warn", crowded+" đang đông nên bạn đứng ngoài cổng. Đi tới cổng để xếp hàng vào nhé.")
		}
		if starter {
			r.toast(p, "good", fmt.Sprintf("🎁 Cụ Cục tặng %d thóc khởi nghiệp. Tiêu cho khéo nhé!", r.hub.bal.StarterThoc))
		}
	}()

	r.discover(p, "zone:"+p.zone)
	r.onJoinNPC(p, now)
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
		"self": r.publicOf(p), "bus": r.busInfo(), "region": r.Region, "actors": r.actorList(), "clock": GameClock(now),
	}
	if r.world != nil {
		welcome["world"] = r.world.WorldInfo()
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
	if r.show != nil {
		p.sess.Send(r.show.msg(now)) // vào giữa trận biểu diễn: xem tiếp từ đúng giây đang diễn
	}
	r.sendPKBoard(p)
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
	r.abandonDig(p)
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
	} else if p.pkEnd != nil {
		s.Send(p.pkEnd) // M10: trận đã chốt trong lúc mất mạng: nối lại vẫn thấy kết quả
	}
	p.pkEnd = nil
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
	r.fbedsDirty = true
	p.leaving = true // xuống xe lúc rời phòng: không dời ra cổng khu, không báo khám phá khu
	// Rời phòng (đổi kênh, sang vùng, vào/ra nhà, bị thay phiên) = bỏ dở lượt đào/câu: tính hồi đất/hồi câu
	// trước khi lưu. Thiếu dòng này thì đổi kênh rồi đào lại ngay tại chỗ để gieo lại loài giun/cá đã lộ.
	r.abandonDig(p)
	if p.trade != nil {
		r.closeTrade(p.trade, "Bên kia đã rời đi.")
	}
	r.parkVehicle(p, "Người lái đã rời đi, bạn xuống xe an toàn.")
	if p.rideOf != nil {
		r.dismount(p, "")
	}
	if p.carrier != nil {
		r.unboard(p, "")
	}
	r.forgetPlayer(p)
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
	Nest     string            `json:"nest,omitempty"`
	X        float64           `json:"x"`
	Z        float64           `json:"z"`
	RY       float64           `json:"ry"`
	Stage    int               `json:"stage"`
	Carrier  string            `json:"carrier,omitempty"` // đang ngồi xe buýt / xe ôm NPC
	CarSeat  int               `json:"carSeat,omitempty"`
}

// Chỉ gửi thông tin công khai; ví/túi đầy đủ không broadcast (mục 16.3).
func (r *Room) publicOf(p *Player) publicPlayer {
	eq := map[string]string{}
	for k, v := range p.acc.Equipped {
		eq[k] = v
	}
	pp := publicPlayer{PID: p.PID, Name: p.acc.Char.Name, Tag: PlayerTag(p.acc.ID), Char: p.acc.Char, Equipped: eq, Level: p.acc.Level,
		Title: titleName(p.acc.Title), Nest: p.seat, Gone: !p.goneAt.IsZero(), X: p.pos.X, Z: p.pos.Z, RY: p.ry, Stage: p.acc.HungerStage(r.hub.bal), Pet: petOf(p.acc)}
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
	if p.carrier != nil {
		pp.Carrier, pp.CarSeat = p.carrier.ID, p.carrierSeat
	}
	_, pp.Guild, pp.GColor = r.hub.store.Guilds.GuildOf(p.acc.ID)
	return pp
}

// sameView: trên bản đồ liền mọi khu nhìn thấy nhau; nhà riêng chỉ có một khu.
func (r *Room) sameView(a, b string) bool { return r.world != nil || a == b }

func (r *Room) zonePlayers(zone string, except *Player) []publicPlayer {
	out := []publicPlayer{}
	for _, o := range r.players {
		if o != except && r.sameView(o.zone, zone) {
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
		if p != except && r.sameView(p.zone, zone) {
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
		if regen := r.homeRest(p); regen > 0 {
			p.acc.restoreMana(regen * dt / 60) // ngủ ổ trong nhà đã xây: hồi độ no theo vật liệu
		} else if r.nearHomeNest(p) {
			p.acc.TickVitalsAt(dt, b, r.drainFactor(p)*b.ManaRestFactor) // Tổ nhỏ chưa có nhà: nghỉ ổ như ở sảnh
		} else {
			p.acc.TickVitalsAt(dt, b, r.drainFactor(p))
		}
		stage := p.acc.HungerStage(b)
		if stage != p.lastStage || now.Sub(p.vitAt) > time.Second {
			if stage != p.lastStage {
				r.broadcastZone(p.zone, nil, map[string]any{"t": "pstage", "pid": p.PID, "stage": stage})
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
		if p.rideOf != nil || p.carrier != nil {
			continue // khách ngồi theo ghế, vị trí do syncRides / syncCarried đặt
		}
		wantMove := now.Sub(p.inAt) < 300*time.Millisecond && (p.in.X != 0 || p.in.Z != 0) && p.dig == nil && p.fish == nil && mult > 0
		if r.stepStamina(p, now, dt, stage, wantMove) {
			mult *= SprintMult
		} else if now.Before(p.gooseUntil) && p.ride == nil {
			mult *= GooseMult
		}
		if p.dig != nil && now.Sub(p.dig.TugAt).Seconds() > p.dig.Params.MaxSec+20 {
			r.abandonDig(p) // lượt đào bỏ dở quá lâu: huỷ để gà đi lại được
		}
		if p.fish != nil && now.Sub(p.fish.TugAt).Seconds() > p.fish.Params.MaxSec+20 {
			r.abandonFish(p)
		}
		if wantMove {
			spd := WalkSpeed * SpeedFactor(stage) * r.walkBonus(p) * mult * math.Min(dt, .1)
			z := r.zones[p.zone]
			if r.world != nil {
				z = r.world.World
			}
			if p.seat != "" {
				r.stand(p)
			}
			next := z.Move(p.pos, Vec{p.pos.X + p.in.X*spd, p.pos.Z + p.in.Z*spd})
			if r.world != nil && p.ride != nil && !DriveZones[r.world.AreaAt(next)] {
				next = p.pos // xe dừng ở mép khu cấm lái (sảnh, vườn, sân PK, trạm ấp)
			}
			if target := r.areaAtOrEmpty(next); target != "" && r.areaFull(p, target) {
				next = p.pos // M9: khu đã đủ người (sảnh 50, khu khác 25) thì dừng ở cổng, vào hàng chờ
				r.sendAreaFull(p, target, now)
				if now.Sub(p.crowdToastAt) > 5*time.Second {
					p.crowdToastAt = now
					r.toast(p, "warn", fmt.Sprintf("Khu này đang đông, bạn đứng thứ %d trong hàng chờ.", r.areaQueuePos(p, target)))
				}
			}
			if r.world != nil && next != p.pos { // nhìn trước 1,5 ô: báo client trước khi nó kịp đoán bước vào khu đầy
				ahead := z.Move(next, Vec{next.X + p.in.X*areaLookahead, next.Z + p.in.Z*areaLookahead})
				if target := r.areaAtOrEmpty(ahead); target != "" && r.areaWouldBlock(p, target) {
					r.sendAreaFull(p, target, now)
				}
			}
			p.pos = next
			p.ry = math.Atan2(p.in.X, p.in.Z)
			p.moving = true
			r.enterArea(p)
		} else {
			p.moving = false
		}
	}
	r.stepPK(now)
	r.stepPKBoard(now)
	r.stepV11(now)
	r.stepActors(now, dt)
	if r.tick%TickRate == 0 {
		r.clockTick(now) // mọi phòng, kể cả nhà riêng, cùng một trời
		r.stepAreaQueues(now)
		if r.tick%(TickRate*10) == 0 {
			r.fbedsDirty = true // kết bạn/huỷ bạn đổi danh sách cây nhìn thấy
		}
		r.syncFriendBeds()
		if r.tick%(TickRate*5) == 0 || r.tick == TickRate {
			r.syncShowcase(now)
		}
	}
	if r.home != nil && r.home.Tick(now) {
		for _, p := range r.players {
			if p.acc.ID == r.Owner {
				p.acc.HouseBuilt, p.dirty = true, true
			}
		}
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
				if p.running {
					mv = 2 // dáng chạy cho người khác thấy
				}
			}
			key := p.zone
			if r.world != nil {
				key = r.world.Outdoor
			}
			byZone[key] = append(byZone[key], []float64{round2(p.pos.X), round2(p.pos.Z), round2(p.ry), mv})
			pidsByZone[key] = append(pidsByZone[key], p.PID)
		}
		var arows [][]any
		for _, a := range r.actors {
			mv := 0
			if a.moving {
				mv = 1
			}
			arows = append(arows, []any{a.ID, round2(a.Pos.X), round2(a.Pos.Z), round2(a.RY), mv, a.State})
		}
		for zone, rows := range byZone {
			msg := map[string]any{"t": "snap", "tick": r.tick, "pids": pidsByZone[zone], "p": rows}
			if arows != nil {
				msg["a"] = arows
			}
			r.broadcastZone(zone, nil, msg)
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

// abandonDig bỏ lượt đào đang mở như thể giun thoát: không thưởng, vẫn tính hồi đất. Nhờ vậy không thể
// bắt đầu lại (huỷ, rời khu, mất kết nối) để gieo lại loài giun đã lộ.
func (r *Room) abandonDig(p *Player) {
	r.abandonFish(p)
	d := p.dig
	if d == nil {
		return
	}
	p.dig = nil
	cd := 1.0
	if d.Exhausted {
		cd = 1.5
	}
	p.acc.DigReadyAt = time.Now().Add(time.Duration(float64(r.hub.bal.DigCooldown) * cd))
	p.dirty = true
	r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "escape"})
}

// nearHomeNest: đứng yên cạnh ổ rơm trong nhà riêng (kể cả Tổ nhỏ chưa xây nhà).
func (r *Room) nearHomeNest(p *Player) bool {
	if r.home == nil || p.moving || p.seat != "" {
		return false
	}
	for _, o := range r.home.Objects {
		if o.Kind == "o_rom" && p.pos.Dist(Vec{float64(o.X), float64(o.Z)}) < 1.3 {
			return true
		}
	}
	return false
}

// homeRest: đứng yên cạnh ổ rơm trong nhà đã xây xong → độ no hồi theo vật liệu nhà (điểm/phút).
func (r *Room) homeRest(p *Player) float64 {
	if r.home == nil || p.moving {
		return 0
	}
	regen, _ := r.home.comfort()
	if regen <= 0 {
		return 0
	}
	for _, o := range r.home.Objects {
		if o.Kind == "o_rom" && p.pos.Dist(Vec{float64(o.X), float64(o.Z)}) < 1.3 {
			return regen
		}
	}
	return 0
}

// drainFactor: ngồi ổ thì độ no tụt chậm; bạn đồng hành "Bụng Khoẻ" giảm thêm.
func (r *Room) drainFactor(p *Player) float64 {
	f := 1.0
	if p.seat != "" {
		f *= r.hub.bal.ManaRestFactor
	}
	return f * (1 - petBonus(p.acc, AbilityBelly))
}

// walkBonus: bạn đồng hành "Chân Nhanh" (tổng mọi nguồn tối đa +10%).
func (r *Room) walkBonus(p *Player) float64 { return 1 + math.Min(0.10, petBonus(p.acc, AbilityFeet)) }

// maxStamina: 6 giây chạy, đói (giai đoạn 2) còn một nửa, đi chậm/kiệt sức không chạy; ngỗng cõng thêm giây.
func (r *Room) maxStamina(p *Player, stage int) float64 {
	if stage >= 3 {
		return 0
	}
	m := StaminaSec + petBonus(p.acc, AbilityCarry)
	if stage == 2 {
		m /= 2
	}
	return m
}

// stepStamina trả true nếu tick này người chơi được chạy nhanh; cập nhật thể lực và báo cho chủ phiên.
func (r *Room) stepStamina(p *Player, now time.Time, dt float64, stage int, moving bool) bool {
	maxS := r.maxStamina(p, stage)
	if !p.stamInit {
		p.stam, p.stamInit = maxS, true
	}
	if !p.stamLock.IsZero() && !now.Before(p.stamLock) {
		p.stam, p.stamLock = maxS, time.Time{} // hết 5 giây hụt hơi: thể lực đầy lại
	}
	canRun := p.run && moving && maxS > 0 && p.ride == nil && p.rideOf == nil && p.seat == "" && p.trade == nil && p.stamLock.IsZero() && p.stam > 0
	p.running = canRun
	if canRun {
		p.stam -= dt
		p.stamUsedAt = now
		if p.stam <= 0 {
			p.stam = 0
			p.stamLock = now.Add(StaminaLock) // hụt hơi: nghỉ 5 giây
			p.memo.lockouts = append(p.memo.lockouts, now)
		}
		if !p.acc.Tutorial["first_run"] {
			p.acc.step("first_run")
			p.dirty = true
		}
	} else if p.stamLock.IsZero() && now.Sub(p.stamUsedAt) > StaminaRegenGap {
		p.stam += maxS * StaminaRegen * dt
	}
	p.stam = math.Min(p.stam, maxS)
	ratio := 0.0
	if maxS > 0 {
		ratio = math.Round(p.stam/maxS*100) / 100
	}
	if p.sess != nil && ratio != p.stamSent && now.Sub(p.stamSentAt) > 150*time.Millisecond {
		lock := 0.0
		if !p.stamLock.IsZero() {
			lock = math.Max(0, p.stamLock.Sub(now).Seconds())
		}
		p.sess.Send(map[string]any{"t": "stam", "s": ratio, "lock": math.Round(lock*10) / 10, "max": maxS})
		p.stamSent, p.stamSentAt = ratio, now
	}
	return canRun
}

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
		p.run = m.Run
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
	Species  string `json:"species"`
}

func petOf(a *Account) *petView {
	c := a.companion(a.Following)
	if c == nil {
		return nil
	}
	lv, _ := AffinityLevel(c.Affinity)
	return &petView{c.Name, c.Color, lv, c.Species}
}
