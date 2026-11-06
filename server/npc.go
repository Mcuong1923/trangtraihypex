package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Làng sống (PLAN_LOGIC_GAME mục 5–9): NPC cố định chào hỏi, diễn viên di động (xe buýt, xe ôm, xe NPC,
// cameo), lời thoại theo hạn mức. Máy chủ quyết định ai nói gì với ai; client chỉ dựng mô hình và hiện bong bóng.
// NPC không bao giờ tính là người chơi: không vào số người online, không có hồ sơ, không nhận tin nhắn.

// Prefs là cài đặt NPC của người chơi.
type Prefs struct {
	Chatter string `json:"chatter"`           // full | normal | low (rỗng = full)
	NoTease bool   `json:"noTease,omitempty"` // không cho NPC trêu mình
}

const (
	KindBus     = "bus"
	KindXeOm    = "xeom"
	KindTraffic = "traffic"
	KindCameo   = "cameo"

	BusCount     = 3
	BusSeatsLoop = 12
	XeOmSpeed    = 8.5
	XeOmCruise   = 6.0
	XeOmShortM   = 90.0 // ≤ 90 m theo đường: 1 thóc, xa hơn 2 thóc
	OfferTTL     = 15 * time.Second
	SayRadius    = 30.0
	SayPerSec    = 3 // câu NPC không nhắm ai, toàn phòng, mỗi giây
	TeasePerHour = 6

	ShipperTipsPerDay = 5 // chỉ đường đúng cho shipper: +1 thóc, tối đa 5 lần/ngày/tài khoản
)

type cue struct {
	At     float64 // giây từ lúc cameo bắt đầu (hoặc mét trên đường đi nếu ByDist)
	Who    string  // id actor nói
	Key    string
	Text   string
	To     *Player
	Choice []string // câu hỏi có lựa chọn (shipper, nhà báo, bán rong)
	done   bool
}

type Actor struct {
	ID    string
	Kind  string
	Role  string // vai cụ thể: tai_xe, xe_om, dap_xe, shipper, van_truot, duong_sinh, thieu_gia, me_ga, ga_con…
	Name  string
	Ride  string            // bus | xe_may | xe_dap | van_truot | o_to | walk
	Look  map[string]string // màu lông, áo, mũ để client dựng mô hình
	Pos   Vec
	RY    float64
	Speed float64
	Path  []Vec
	PI    int
	State string
	Until time.Time
	Born  time.Time
	// Xe buýt
	S       float64
	StopIdx int
	Seats   []*Player
	// Xe ôm
	Target  *Player
	Dest    string
	Fare    int
	OfferID string
	Stand   bool // xe đậu chờ ở bến xe ôm cây đa
	// Cameo
	Cues    []*cue
	DistRun float64 // mét đã chạy trên đường đi cameo
	Slow    [2]float64
	saidAt  time.Time
	moving  bool
}

type actorInfo struct {
	ID   string            `json:"id"`
	Kind string            `json:"kind"`
	Role string            `json:"role"`
	Name string            `json:"name"`
	Ride string            `json:"ride"`
	Look map[string]string `json:"look,omitempty"`
}

func (a *Actor) info() actorInfo {
	return actorInfo{a.ID, a.Kind, a.Role, a.Name, a.Ride, a.Look}
}

// npcMemo: trí nhớ ngắn của đạo diễn về từng người chơi (chỉ trong phiên).
type npcMemo struct {
	offerAt    time.Time
	declines   []time.Time
	teaseAt    time.Time
	teases     []time.Time // các lần bị trêu trong giờ qua
	buddySince time.Time   // bắt đầu ngồi ổ cạnh bạn bè
	cameoAt    time.Time
	greetAt    map[string]time.Time
	lockouts   []time.Time
	stillAt    time.Time
	stillPos   Vec
	lostPKAt   time.Time
	reactAt    time.Time
	hungryAt   time.Time
	markSent   string
	welcomeAt  time.Time // Cụ Cục gọi gà mới / chào sau điểm danh
	checkinDay int
}

// ---------- Khởi tạo & nhịp ----------

func (r *Room) roads() *RoadNet {
	if r.world == nil || r.world.Roads == nil {
		return nil
	}
	return r.world.Roads
}

func (r *Room) newActor(kind, role, name, ride string, look map[string]string, pos Vec, now time.Time) *Actor {
	if r.actors == nil {
		r.actors = map[string]*Actor{}
	}
	r.actorSeq++
	a := &Actor{ID: "a" + strconv.Itoa(r.actorSeq), Kind: kind, Role: role, Name: name, Ride: ride, Look: look, Pos: pos, Born: now, StopIdx: -1}
	r.actors[a.ID] = a
	r.broadcastRoom(nil, map[string]any{"t": "aj", "a": a.info()})
	return a
}

func (r *Room) dropActor(a *Actor) {
	for _, p := range a.Seats {
		if p != nil {
			r.unboard(p, "")
		}
	}
	delete(r.actors, a.ID)
	r.broadcastRoom(nil, map[string]any{"t": "al", "id": a.ID})
}

func (r *Room) actorList() []actorInfo {
	out := []actorInfo{}
	for _, a := range r.actors {
		out = append(out, a.info())
	}
	return out
}

func (r *Room) rnd() float64 { return r.rng.Float64() }

// initActors dựng xe buýt và xe NPC cho chuồng công cộng có đường làng (chỉ Hypex có vành đai).
func (r *Room) initActors(now time.Time) {
	rn := r.roads()
	r.actorsReady = true
	if rn == nil || r.Region != RegionHypex || rn.BusLen() == 0 {
		return
	}
	drivers := []string{"Bác Tài Gáy", "Chú Tài Mào Đỏ", "Cô Tài Lông Mượt"}
	for i := 0; i < BusCount; i++ {
		a := r.newActor(KindBus, "tai_xe", drivers[i%len(drivers)], "bus", map[string]string{"color": []string{"nau", "trang", "vang"}[i%3]}, Vec{}, now)
		a.S = rn.BusLen() * float64(i) / BusCount
		a.Seats = make([]*Player, BusSeatsLoop)
		a.Pos, a.RY = rn.BusAt(a.S)
		a.State = "drive"
	}
	r.spawnTraffic(now, GameClock(now).Night)
	// Xe ôm đậu chờ ở bến cây đa.
	stand := r.xeomStandPos()
	s := r.newActor(KindXeOm, "xe_om", "Anh Ba Xe Ôm", "xe_may", map[string]string{"color": "nau"}, stand, now)
	s.Stand, s.State, s.Seats = true, "stand", make([]*Player, 1)
	s.RY = math.Pi / 2
}

// Giao thông: 3 xe đạp, 2 xe máy, 2 ván trượt, 1 cụ tập dưỡng sinh (PLAN_LOGIC_GAME 9.1). Ban đêm chỉ còn
// shipper chạy đơn, những người khác về nhà (PLAN_LOGIC_GAME 16).
var trafficRoles = []struct{ role, name, ride string }{
	{"dap_xe", "Gà Đạp Xe Đi Chợ", "xe_dap"}, {"dap_xe", "Gà Đạp Xe Đi Chợ", "xe_dap"}, {"dap_xe", "Gà Đạp Xe Đi Chợ", "xe_dap"},
	{"shipper", "Gà Shipper", "xe_may"}, {"shipper", "Gà Shipper", "xe_may"},
	{"van_truot", "Gà Teen", "van_truot"}, {"van_truot", "Gà Teen", "van_truot"},
	{"duong_sinh", "Cụ Gà Tập Dưỡng Sinh", "walk"},
}

func nightShift(role string) bool { return role == "shipper" }

// spawnTraffic thả các xe chưa có trên đường: ban ngày đủ cả, ban đêm chỉ ca đêm.
func (r *Room) spawnTraffic(now time.Time, night bool) {
	rn := r.roads()
	have := map[string]int{}
	for _, a := range r.actors {
		if a.Kind == KindTraffic {
			have[a.Role]++
		}
	}
	speeds := map[string]float64{"xe_dap": 4.6, "xe_may": 7, "van_truot": 5.4, "walk": 1.7}
	colors := []string{"vang", "kem", "nau", "trang", "cam", "xam"}
	for i, ro := range trafficRoles {
		if night && !nightShift(ro.role) {
			continue
		}
		if have[ro.role] > 0 {
			have[ro.role]--
			continue
		}
		start := rn.Nodes[int(r.rnd()*float64(len(rn.Nodes)))%len(rn.Nodes)]
		a := r.newActor(KindTraffic, ro.role, ro.name, ro.ride, map[string]string{"color": colors[i%len(colors)]}, start, now)
		a.Speed = speeds[ro.ride] * (0.9 + r.rnd()*0.2)
		a.Path = rn.RandomEdgePath(start, 6, r.rnd)
		a.State = "cruise"
	}
}

// trafficForClock: trời tối thì người đi chơi về nhà, trời sáng thì ra đường lại.
func (r *Room) trafficForClock(night bool, now time.Time) {
	if !r.actorsReady || r.roads() == nil || r.Region != RegionHypex {
		return
	}
	if night {
		for _, a := range r.actors {
			if a.Kind == KindTraffic && !nightShift(a.Role) {
				r.dropActor(a)
			}
		}
		return
	}
	r.spawnTraffic(now, false)
}

func (r *Room) xeomStandPos() Vec {
	h := r.world.Hub
	return Vec{h.X - 6.5, h.Z + 9.5}
}

// stepActors chạy mỗi tick: di chuyển diễn viên, đặt khách theo ghế. Đạo diễn chạy 2 lần/giây.
func (r *Room) stepActors(now time.Time, dt float64) {
	if r.Kind != "public" || r.world == nil {
		return
	}
	if !r.actorsReady {
		r.initActors(now)
	}
	for _, a := range r.actors {
		switch a.Kind {
		case KindBus:
			r.stepBus(a, now, dt)
		case KindXeOm:
			r.stepXeOm(a, now, dt)
		case KindTraffic:
			r.stepTraffic(a, now, dt)
		case KindCameo:
			r.stepCameo(a, now, dt)
		}
	}
	r.syncCarried()
	if r.tick%10 == 0 {
		r.director(now)
	}
	if r.tick%(TickRate*5) == 0 {
		r.balanceXeOm(now)
	}
}

// follow đi theo đường gấp khúc; trả true khi tới cuối.
func (a *Actor) follow(dt, speed float64) bool {
	a.moving = false
	for a.PI < len(a.Path)-1 {
		next := a.Path[a.PI+1]
		d := a.Pos.Dist(next)
		step := speed * dt
		if d > 1e-6 {
			a.RY = math.Atan2(next.X-a.Pos.X, next.Z-a.Pos.Z)
		}
		if step < d {
			a.Pos = Vec{a.Pos.X + (next.X-a.Pos.X)/d*step, a.Pos.Z + (next.Z-a.Pos.Z)/d*step}
			a.moving = true
			a.DistRun += step
			return false
		}
		a.Pos = next
		a.DistRun += d
		a.PI++
		dt -= d / speed
		if dt <= 0 {
			a.moving = true
			return a.PI >= len(a.Path)-1
		}
	}
	return true
}

func (r *Room) actorSay(a *Actor, key string, kv map[string]string, to *Player) {
	lines := NPCLines[key]
	if len(lines) == 0 {
		return
	}
	text := fillLine(lines[int(r.rnd()*float64(len(lines)))%len(lines)], kv)
	r.say(a.ID, a.Name, text, a.Pos, to)
	a.saidAt = time.Now()
}

// say gửi bong bóng thoại cho người chơi trong bán kính quanh chỗ nói. to != nil: câu nói với riêng người đó
// (họ thấy viền cam và dòng nhật ký); người xung quanh vẫn thấy bong bóng.
func (r *Room) say(id, name, text string, at Vec, to *Player) {
	// Toàn phòng tối đa SayPerSec câu/giây (PLAN_LOGIC_GAME 5.2): câu tán gẫu không nhắm ai bị bỏ khi quá mức;
	// câu nói với một người (hướng dẫn, mời xe, cameo) đã có hạn mức riêng theo người nên luôn đi.
	if now := time.Now(); now.Sub(r.sayWin) >= time.Second {
		r.sayWin, r.sayN = now, 0
	}
	if to == nil && r.sayN >= SayPerSec {
		return
	}
	r.sayN++
	msg := map[string]any{"t": "say", "id": id, "name": name, "text": text, "ttl": math.Min(9, 2.5+float64(len([]rune(text)))/18)}
	if to != nil {
		msg["to"] = to.PID
	}
	for _, p := range r.players {
		if p == to || p.pos.Dist(at) <= SayRadius {
			if p != to && p.acc.Prefs.Chatter == "low" {
				continue
			}
			p.sess.Send(msg)
		}
	}
}

// sayFrom: NPC cố định (theo mã item) nói. Người ở xa nhận dạng thư báo (remote) thay vì bong bóng.
func (r *Room) sayFrom(itemID, key string, kv map[string]string, to *Player) {
	lines := NPCLines[key]
	if len(lines) == 0 {
		return
	}
	text := fillLine(lines[int(r.rnd()*float64(len(lines)))%len(lines)], kv)
	var it *Item
	for _, z := range r.zones {
		if x := z.Item(itemID); x != nil {
			it = x
			break
		}
	}
	name := itemID
	at := Vec{}
	if it != nil {
		name, at = strings.Split(it.Label, " (")[0], it.Pos()
		name = strings.Split(name, " · ")[0]
	}
	// Người cùng khu trong 50 m vẫn thấy bong bóng trên đầu NPC (Cụ Cục gọi từ bục cuối sảnh); xa hơn thì nhận thư.
	if to != nil && (it == nil || to.pos.Dist(at) > 50 || to.pos.Dist(at) > SayRadius && r.world != nil && r.world.AreaAt(at) != to.zone) {
		to.sess.Send(map[string]any{"t": "say", "id": itemID, "name": name, "text": text, "to": to.PID, "remote": true, "ttl": 6})
		return
	}
	r.say(itemID, name, text, at, to)
}

func kvFor(p *Player) map[string]string {
	return map[string]string{"ten": p.acc.Char.Name, "anh": pronoun(p.acc), "buoi": daypart(time.Now())}
}

// ---------- Xe buýt vòng làng (PLAN_LOGIC_GAME 7) ----------

func (r *Room) stepBus(a *Actor, now time.Time, dt float64) {
	rn := r.roads()
	L := rn.BusLen()
	if a.State == "dwell" {
		if now.Before(a.Until) {
			return
		}
		a.State, a.StopIdx = "drive", -1
		if r.rnd() < 0.3 {
			r.actorSay(a, "bus.depart", nil, nil)
		}
	}
	idx, dist := r.nextStop(a.S)
	step := r.hub.bal.BusLoopSpeed * dt
	if step >= dist {
		a.S = rn.Stops[idx].S
		a.State, a.StopIdx, a.Until = "dwell", idx, now.Add(r.hub.bal.BusLoopDwell)
		a.Pos, a.RY = rn.BusAt(a.S - 0.5)
		a.Pos, _ = rn.BusAt(a.S)
		r.busArrive(a, idx)
		a.moving = false
		return
	}
	a.S = math.Mod(a.S+step, L)
	a.Pos, a.RY = rn.BusAt(a.S)
	a.moving = true
}

// nextStop: bến kế tiếp phía trước vị trí s và quãng đường tới đó.
func (r *Room) nextStop(s float64) (int, float64) {
	rn := r.roads()
	L := rn.BusLen()
	best, bd := 0, math.MaxFloat64
	for i, st := range rn.Stops {
		d := math.Mod(st.S-s+L, L)
		if d < 1e-6 {
			d = L
		}
		if d < bd {
			best, bd = i, d
		}
	}
	return best, bd
}

// busETA: giây tới khi một xe buýt dừng ở bến idx (0 nếu đang có xe dừng).
func (r *Room) busETA(idx int, now time.Time) (float64, *Actor) {
	rn := r.roads()
	L := rn.BusLen()
	best, bestBus := math.MaxFloat64, (*Actor)(nil)
	for _, a := range r.actors {
		if a.Kind != KindBus {
			continue
		}
		if a.State == "dwell" && a.StopIdx == idx {
			return 0, a
		}
		d := math.Mod(rn.Stops[idx].S-a.S+L, L)
		stops := 0
		for _, st := range rn.Stops {
			if x := math.Mod(st.S-a.S+L, L); x > 1e-6 && x < d {
				stops++
			}
		}
		eta := d/r.hub.bal.BusLoopSpeed + float64(stops)*r.hub.bal.BusLoopDwell.Seconds()
		if a.State == "dwell" {
			eta += math.Max(0, a.Until.Sub(now).Seconds())
		}
		if eta < best {
			best, bestBus = eta, a
		}
	}
	return best, bestBus
}

func (r *Room) busArrive(a *Actor, idx int) {
	st := r.roads().Stops[idx]
	alighted := false
	for _, p := range a.Seats {
		if p != nil && (p.busDest == st.ID || p.alightNext) {
			r.unboard(p, "🚌 Tới bến "+st.Title+"!")
			alighted = true
		}
	}
	near := false
	for _, p := range r.players {
		if p.pos.Dist(st.Pos) < 12 {
			near = true
		}
	}
	if (alighted || near) && r.rnd() < 0.7 {
		r.actorSay(a, "bus.arrive", map[string]string{"khu": st.Title}, nil)
	}
}

func (r *Room) stopAt(p *Player) int {
	rn := r.roads()
	if rn == nil {
		return -1
	}
	for i, st := range rn.Stops {
		if p.pos.Dist(st.Pos) < 5 {
			return i
		}
	}
	return -1
}

func (r *Room) busy(p *Player) error {
	switch {
	case p.carrier != nil:
		return errors.New("Bạn đang ngồi xe rồi.")
	case p.ride != nil:
		return errors.New("Cất xe riêng ở điểm đỗ trước đã.")
	case p.rideOf != nil:
		return errors.New("Đang ngồi xe của người khác; xuống xe trước.")
	case p.match != nil:
		return errors.New("Đang trong trận PK.")
	case p.dig != nil:
		return errors.New("Đang kéo giun.")
	case p.trade != nil:
		return errors.New("Đang giao dịch.")
	}
	return nil
}

func seatOf(a *Actor) int {
	for i, q := range a.Seats {
		if q == nil {
			return i
		}
	}
	return -1
}

func (r *Room) board(p *Player, a *Actor, seat int) {
	a.Seats[seat] = p
	p.carrier, p.carrierSeat = a, seat
	if p.seat != "" {
		r.stand(p)
	}
	p.in = Vec{}
	r.broadcastRoom(nil, map[string]any{"t": "pcar", "pid": p.PID, "carrier": a.ID, "seat": seat, "kind": a.Kind})
}

// unboard cho khách xuống cạnh xe, ở chỗ đứng được gần nhất.
func (r *Room) unboard(p *Player, why string) {
	a := p.carrier
	if a == nil {
		return
	}
	for i, q := range a.Seats {
		if q == p {
			a.Seats[i] = nil
		}
	}
	p.carrier, p.busDest, p.alightNext = nil, "", false
	spot := Vec{a.Pos.X + math.Cos(a.RY)*1.8, a.Pos.Z - math.Sin(a.RY)*1.8}
	if a.Kind == KindBus && a.StopIdx >= 0 {
		spot = r.roads().Stops[a.StopIdx].Pos
	}
	if !r.world.World.Walkable(spot) {
		spot = a.Pos
	}
	p.pos = spot
	kind := "good"
	if t := r.keepOutOfFullArea(p); t != "" {
		kind, why = "warn", t+" đang đông, bạn xuống xe ở ngoài cổng."
	}
	r.enterArea(p)
	r.broadcastRoom(nil, map[string]any{"t": "pcar", "pid": p.PID, "carrier": ""})
	if why != "" {
		r.toast(p, kind, why)
	}
}

// syncCarried đặt khách theo ghế của xe NPC (như syncRides cho xe người chơi).
func (r *Room) syncCarried() {
	for _, a := range r.actors {
		if len(a.Seats) == 0 {
			continue
		}
		s, c := math.Sin(a.RY), math.Cos(a.RY)
		for i, q := range a.Seats {
			if q == nil {
				continue
			}
			var o Vec
			if a.Kind == KindBus {
				o = Vec{-0.55 + 1.1*float64(i%2), 2.1 - 0.95*float64(i/2)}
			} else {
				o = Vec{0, -0.75}
			}
			q.pos = Vec{a.Pos.X + o.X*c + o.Z*s, a.Pos.Z - o.X*s + o.Z*c}
			q.ry = a.RY
			q.moving = a.moving
			r.enterArea(q)
		}
	}
}

// ---------- Xe ôm (PLAN_LOGIC_GAME 8) ----------

func (r *Room) outdoorWalkers() []*Player {
	var out []*Player
	for _, p := range r.players {
		if p.goneAt.IsZero() && p.zone == r.world.Outdoor && p.carrier == nil && p.ride == nil && p.rideOf == nil {
			out = append(out, p)
		}
	}
	return out
}

// balanceXeOm giữ số xe ôm chạy rong = clamp(1 + ceil(người ngoài trời / 2), 2, 6).
func (r *Room) balanceXeOm(now time.Time) {
	rn := r.roads()
	if rn == nil || r.Region != RegionHypex {
		return
	}
	want := min(6, max(2, 1+int(math.Ceil(float64(len(r.outdoorWalkers()))/2))))
	var cruising []*Actor
	for _, a := range r.actors {
		if a.Kind == KindXeOm && !a.Stand {
			cruising = append(cruising, a)
		}
	}
	names := []string{"Anh Tư Bốc Đầu", "Chú Sáu Tay Lái Lụa", "Cô Ba Xe Ôm Công Nghệ", "Anh Bảy Đường Quen", "Chị Năm Phanh Gấp", "Chú Út Đội Mũ Xanh"}
	for len(cruising) < want {
		// Sinh ở nút đường xa các xe ôm khác nhất để xe rải đều.
		best, bd := Vec{}, -1.0
		for i := 0; i < 12; i++ {
			c := rn.Nodes[int(r.rnd()*float64(len(rn.Nodes)))%len(rn.Nodes)]
			d := 1e9
			for _, o := range cruising {
				d = math.Min(d, c.Dist(o.Pos))
			}
			if d > bd {
				best, bd = c, d
			}
		}
		a := r.newActor(KindXeOm, "xe_om", names[len(cruising)%len(names)], "xe_may", map[string]string{"color": []string{"nau", "vang", "xam", "kem"}[len(cruising)%4]}, best, now)
		a.State, a.Seats = "cruise", make([]*Player, 1)
		a.Path = rn.RandomEdgePath(best, 5, r.rnd)
		cruising = append(cruising, a)
	}
	for len(cruising) > want {
		a := cruising[len(cruising)-1]
		cruising = cruising[:len(cruising)-1]
		if a.State == "cruise" || a.State == "rest" {
			r.dropActor(a)
		}
	}
}

func (r *Room) stepXeOm(a *Actor, now time.Time, dt float64) {
	rn := r.roads()
	switch a.State {
	case "stand":
		a.moving = false
	case "cruise":
		if a.follow(dt, XeOmCruise) {
			a.Path, a.PI = rn.RandomEdgePath(a.Pos, 5, r.rnd), 0
		}
	case "approach":
		t := a.Target
		if t == nil || t.goneAt.After(time.Time{}) || t.carrier != nil || t.pos.Dist(a.Pos) > 60 {
			r.xeomRelease(a, false)
			return
		}
		goal := Vec{t.pos.X + math.Cos(t.ry)*2.4, t.pos.Z - math.Sin(t.ry)*2.4}
		if !r.world.World.Walkable(goal) {
			goal = t.pos
		}
		a.Path, a.PI = []Vec{a.Pos, goal}, 0
		if a.Pos.Dist(goal) < 0.6 || a.follow(dt, 7) {
			a.State, a.Until = "offer", now.Add(OfferTTL)
			a.OfferID = fmt.Sprintf("of%d", now.UnixNano())
			a.RY = math.Atan2(t.pos.X-a.Pos.X, t.pos.Z-a.Pos.Z)
			t.memo.offerAt = now
			r.actorSay(a, "xeom.offer", kvFor(t), t)
			t.sess.Send(map[string]any{"t": "offer", "id": a.OfferID, "actor": a.ID, "kind": "xeom", "driver": a.Name, "ttl": OfferTTL.Seconds(),
				"short": 1, "long": 2, "shortM": XeOmShortM})
		}
	case "offer":
		a.moving = false
		t := a.Target
		if t == nil || now.After(a.Until) || t.pos.Dist(a.Pos) > 9 || t.carrier != nil || !t.goneAt.IsZero() {
			r.xeomRelease(a, false)
		}
	case "carry":
		if a.follow(dt, XeOmSpeed) {
			a.State, a.Until = "drop", now.Add(1500*time.Millisecond)
			if p := a.Seats[0]; p != nil {
				r.unboard(p, "🛵 Tới "+zoneTitle(a.Dest)+"!")
				r.actorSay(a, "xeom.thanks", kvFor(p), p)
			}
		} else if r.rnd() < dt/9 && now.Sub(a.saidAt) > 8*time.Second {
			r.actorSay(a, "xeom.ride", nil, a.Seats[0])
		}
	case "drop":
		a.moving = false
		if now.After(a.Until) {
			a.State, a.Until, a.Target = "rest", now.Add(20*time.Second), nil
		}
	case "rest":
		a.moving = false
		if now.After(a.Until) {
			a.State, a.Path, a.PI = "cruise", rn.RandomEdgePath(a.Pos, 5, r.rnd), 0
		}
	}
}

// xeomRelease bỏ khách đang mời; declined=true khi khách bấm "Không".
func (r *Room) xeomRelease(a *Actor, declined bool) {
	if t := a.Target; t != nil {
		t.sess.Send(map[string]any{"t": "offer_end", "id": a.OfferID})
		if declined {
			t.memo.declines = append(t.memo.declines, time.Now())
			r.actorSay(a, "xeom.decline", kvFor(t), t)
		}
	}
	a.Target, a.OfferID = nil, ""
	if a.Stand {
		a.State, a.Pos = "stand", r.xeomStandPos()
		return
	}
	a.State, a.Path, a.PI = "cruise", r.roads().RandomEdgePath(a.Pos, 5, r.rnd), 0
}

func (r *Room) offerAllowed(p *Player, now time.Time) bool {
	if p.acc.Thoc < 1 || now.Sub(p.joinedAt) < 2*time.Minute && !p.acc.TutorialClaimed && !p.acc.Tutorial["first_sell"] {
		return false
	}
	if now.Sub(p.memo.offerAt) < 3*time.Minute {
		return false
	}
	recent := 0
	for _, d := range p.memo.declines {
		if now.Sub(d) < 10*time.Minute {
			recent++
		}
	}
	if recent >= 2 && now.Sub(p.memo.declines[len(p.memo.declines)-1]) < 15*time.Minute {
		return false
	}
	return p.dig == nil && p.trade == nil && p.match == nil && p.seat == ""
}

// xeomFare: 1 thóc nếu ≤ 90 m theo đường, 2 thóc nếu xa hơn; "Mặc Cả" giảm nhưng tối thiểu 1. Trời mưa phụ
// thu 1 thóc khăn lau (PLAN_LOGIC_GAME 16).
func xeomFare(a *Account, meters float64, now time.Time) int {
	fare := 1
	if meters > XeOmShortM {
		fare = 2
	}
	fare = max(1, fareAfter(a, fare))
	if GameClock(now).Rain {
		fare += RainXeOmFare
	}
	return fare
}

// ---------- Xe NPC trên đường (PLAN_LOGIC_GAME 9.1) ----------

func (r *Room) stepTraffic(a *Actor, now time.Time, dt float64) {
	speed := a.Speed
	// Có người chơi chắn phía trước trong 3 m thì giảm tốc, thỉnh thoảng bóp còi.
	fx, fz := math.Sin(a.RY), math.Cos(a.RY)
	for _, p := range r.players {
		dx, dz := p.pos.X-a.Pos.X, p.pos.Z-a.Pos.Z
		ahead := dx*fx + dz*fz
		if ahead > 0 && ahead < 3 && math.Abs(dx*fz-dz*fx) < 1.2 && p.carrier == nil {
			speed = math.Min(speed, 1)
			if now.Sub(a.saidAt) > 20*time.Second && a.Ride != "walk" {
				r.actorSay(a, "traffic.honk", nil, nil)
			}
		}
	}
	if a.follow(dt, speed) {
		a.Path, a.PI = r.roads().RandomEdgePath(a.Pos, 6, r.rnd), 0
	}
}

// ---------- Cameo nhắm vào người chơi (PLAN_LOGIC_GAME 9.2) ----------

func (r *Room) stepCameo(a *Actor, now time.Time, dt float64) {
	speed := a.Speed
	if a.Slow[1] > 0 && a.DistRun > a.Slow[0] && a.DistRun < a.Slow[1] {
		speed = math.Min(speed, 2.2)
	}
	if a.State == "wait" {
		a.moving = false
		if now.After(a.Until) {
			a.State = "go"
		}
	} else if a.follow(dt, speed) && a.State != "ask" {
		r.dropActor(a)
		return
	}
	el := now.Sub(a.Born).Seconds()
	for _, c := range a.Cues {
		if c.done || (c.At >= 0 && el < c.At) || (c.At < 0 && a.DistRun < -c.At) {
			continue
		}
		c.done = true
		who := a
		if c.Who != "" && r.actors[c.Who] != nil {
			who = r.actors[c.Who]
		}
		text := c.Text
		if text == "" {
			lines := NPCLines[c.Key]
			if len(lines) == 0 {
				continue
			}
			kv := map[string]string{}
			if c.To != nil {
				kv = kvFor(c.To)
			}
			text = fillLine(lines[int(r.rnd()*float64(len(lines)))%len(lines)], kv)
		}
		if len(c.Choice) > 0 && c.To != nil {
			a.State, a.Until = "ask", now.Add(20*time.Second)
			c.To.sess.Send(map[string]any{"t": "npcask", "id": a.ID, "name": who.Name, "text": text, "choices": c.Choice, "ttl": 20})
		}
		r.say(who.ID, who.Name, text, who.Pos, c.To)
	}
	if a.State == "ask" && now.After(a.Until) {
		a.State = "go"
	}
	if now.Sub(a.Born) > 90*time.Second {
		r.dropActor(a)
	}
}

// cameoPath: đường thẳng đi ngang qua người chơi, lệch side mét, từ back mét phía sau tới fwd mét phía trước.
func (r *Room) cameoPath(p *Player, back, fwd, side float64) ([]Vec, bool) {
	for try := 0; try < 6; try++ {
		ang := p.ry + float64(try)*math.Pi/3
		fx, fz := math.Sin(ang), math.Cos(ang)
		rx, rz := math.Cos(ang), -math.Sin(ang)
		a := Vec{p.pos.X - fx*back + rx*side, p.pos.Z - fz*back + rz*side}
		b := Vec{p.pos.X + fx*fwd + rx*side, p.pos.Z + fz*fwd + rz*side}
		ok := true
		for k := 0.0; k <= 1; k += 0.05 {
			q := Vec{a.X + (b.X-a.X)*k, a.Z + (b.Z-a.Z)*k}
			if !r.world.World.Walkable(q) || r.world.AreaAt(q) != r.world.Outdoor {
				ok = false
				break
			}
		}
		if ok {
			return []Vec{a, b}, true
		}
	}
	return nil, false
}

// teaseOK: hạn mức trêu của 5.2: ≤ 1 lần/5 phút, ≤ TeasePerHour lần/giờ, không trêu 5 phút đầu; câu trêu cũng
// là cameo nên còn chịu hạn 1 cameo/8 phút.
func (r *Room) teaseOK(p *Player, now time.Time) bool {
	if p.acc.Prefs.NoTease || p.acc.Prefs.Chatter == "low" || now.Sub(p.memo.teaseAt) <= 5*time.Minute ||
		now.Sub(p.memo.cameoAt) <= 8*time.Minute || now.Sub(p.joinedAt) <= 5*time.Minute {
		return false
	}
	recent := p.memo.teases[:0]
	for _, t := range p.memo.teases {
		if now.Sub(t) < time.Hour {
			recent = append(recent, t)
		}
	}
	p.memo.teases = recent
	return len(recent) < TeasePerHour
}

// teased ghi nhận một lần trêu (cũng là một cameo).
func (p *Player) teased(now time.Time) {
	p.memo.teaseAt, p.memo.cameoAt = now, now
	p.memo.teases = append(p.memo.teases, now)
}

func (r *Room) cameoOK(p *Player, now time.Time) bool {
	return p.acc.Prefs.Chatter != "low" && now.Sub(p.memo.cameoAt) > 8*time.Minute && now.Sub(p.joinedAt) > 2*time.Minute
}

// richKid: thiếu gia đi ô tô chậm lại cạnh người đi bộ, trêu rồi rồ ga phóng đi.
func (r *Room) richKid(p *Player, now time.Time) bool {
	path, ok := r.cameoPath(p, 40, 45, 2.6)
	if !ok {
		return false
	}
	a := r.newActor(KindCameo, "thieu_gia", "Gà Thiếu Gia", "o_to", map[string]string{"color": "trang", "eye": "kinh_ram", "car": "#c8343a"}, path[0], now)
	a.Path, a.Speed = path, 11
	a.Slow = [2]float64{36.5, 41}
	key := "rich.mock"
	switch {
	case p.acc.OwnedVehicles["o_to"]:
		key = "rich.respect"
	case p.acc.OwnedVehicles["xe_dap"] && !p.acc.OwnedVehicles["xe_may"]:
		key = "rich.bike"
	}
	a.Cues = []*cue{{At: -36, Key: key, To: p}}
	if key != "rich.respect" {
		a.Cues = append(a.Cues, &cue{At: -42, Key: "rich.laugh", To: p})
	}
	p.teased(now)
	return true
}

// momAndChick: hai mẹ con gà đi ngang, mẹ dạy con lấy người chơi làm ví dụ.
func (r *Room) momAndChick(p *Player, now time.Time) bool {
	path, ok := r.cameoPath(p, 14, 16, 2.2)
	if !ok {
		return false
	}
	mom := r.newActor(KindCameo, "me_ga", "Gà Mẹ", "walk", map[string]string{"color": "nau", "head": "khan"}, path[0], now)
	mom.Path, mom.Speed = path, 1.7
	off := Vec{0.7, 0.4}
	chick := r.newActor(KindCameo, "ga_con", "Gà Con", "walk", map[string]string{"color": "vang", "scale": "0.5"}, Vec{path[0].X + off.X, path[0].Z + off.Z}, now)
	chick.Path, chick.Speed = []Vec{{path[0].X + off.X, path[0].Z + off.Z}, {path[1].X + off.X, path[1].Z + off.Z}}, 1.7
	mom.Cues = []*cue{{At: -12.5, Key: "mom.lesson", To: p}, {At: -16.5, Who: chick.ID, Key: "chick.reply", To: p}}
	p.teased(now)
	return true
}

func dirName(from, to Vec) string {
	dx, dz := to.X-from.X, to.Z-from.Z
	if math.Abs(dx) > math.Abs(dz) {
		if dx < 0 {
			return "Hướng tây"
		}
		return "Hướng đông"
	}
	if dz < 0 {
		return "Hướng bắc"
	}
	return "Hướng nam"
}

// shipperAsks: Gà Shipper hỏi đường quanh quảng trường; chỉ đúng được 1 thóc (tối đa 5 lần/ngày).
func (r *Room) shipperAsks(p *Player, now time.Time) bool {
	targets := []string{"market", "garden", "station", "hatchery", "arena", "park"}
	dest := targets[int(r.rnd()*float64(len(targets)))%len(targets)]
	z := r.zones[dest]
	if z == nil || z.Entry == nil {
		return false
	}
	path, ok := r.cameoPath(p, 10, 0.1, 1.6)
	if !ok {
		return false
	}
	a := r.newActor(KindCameo, "shipper", "Gà Shipper", "xe_may", map[string]string{"color": "cam", "head": "mu_luoi"}, path[0], now)
	a.Path, a.Speed = path, 5
	right := dirName(r.world.Hub, *z.Entry)
	opts := []string{"Hướng bắc", "Hướng nam", "Hướng đông", "Hướng tây"}
	var choices []string
	choices = append(choices, right)
	for _, o := range opts {
		if o != right && len(choices) < 3 {
			choices = append(choices, o)
		}
	}
	// Xáo vị trí đáp án đúng.
	k := int(r.rnd() * 3)
	choices[0], choices[k] = choices[k], choices[0]
	a.Dest = right
	a.Cues = []*cue{{At: -9.5, Text: fillLine(NPCLines["shipper.ask"][0], map[string]string{"khu": z.Title}), To: p, Choice: choices}}
	a.Target = p
	p.memo.cameoAt = now
	return true
}

// reporter: Gà Nhà Báo chạy tới phỏng vấn sau khi đào được giun từ Hiếm trở lên.
func (r *Room) reporter(p *Player, w *Worm, now time.Time) bool {
	ang := p.ry + math.Pi/2
	start := Vec{p.pos.X + math.Sin(ang)*12, p.pos.Z + math.Cos(ang)*12}
	if !r.world.World.Walkable(start) {
		start = Vec{p.pos.X - math.Sin(ang)*12, p.pos.Z - math.Cos(ang)*12}
		if !r.world.World.Walkable(start) {
			return false
		}
	}
	end := Vec{p.pos.X + (start.X-p.pos.X)*0.15, p.pos.Z + (start.Z-p.pos.Z)*0.15}
	a := r.newActor(KindCameo, "nha_bao", "Gà Nhà Báo", "walk", map[string]string{"color": "trang", "head": "mu_phot", "mic": "1"}, start, now)
	a.Path, a.Speed = []Vec{start, end}, 6
	a.Target, a.Dest = p, w.ID
	a.Cues = []*cue{{At: -9.8, Text: fillLine(NPCLines["reporter.ask"][0], map[string]string{"giun": w.Name}), To: p,
		Choice: []string{"Tay nghề thôi, may mắn chỉ 99%!", "Cảm ơn mẹ, cảm ơn Lợn Ủn!", "Tối nay ăn mừng bằng… giun!"}}}
	p.memo.cameoAt = now
	return true
}

// vendor: Gà Bán Rong mang cám tới tận nơi khi người chơi đói và ở xa chợ (giá +20%).
func (r *Room) vendor(p *Player, now time.Time) bool {
	path, ok := r.cameoPath(p, 12, 0.1, 1.8)
	if !ok {
		return false
	}
	a := r.newActor(KindCameo, "ban_rong", "Gà Bán Rong", "walk", map[string]string{"color": "kem", "head": "mu_rom", "tray": "1"}, path[0], now)
	a.Path, a.Speed = path, 2.6
	a.Target = p
	c, rau := foodByID("cam"), foodByID("rau")
	a.Cues = []*cue{{At: -11.5, Key: "vendor.offer", To: p, Choice: []string{
		fmt.Sprintf("Mua & ăn %s (%d thóc)", c.Name, vendorPrice(c.Cost)), fmt.Sprintf("Mua & ăn %s (%d thóc)", rau.Name, vendorPrice(rau.Cost)), "Thôi, cảm ơn"}}}
	p.memo.cameoAt = now
	return true
}

func vendorPrice(cost int) int { return int(math.Ceil(float64(cost) * 1.2)) }

func (r *Room) elderExercise(p *Player, now time.Time) bool {
	path, ok := r.cameoPath(p, 9, 9, 2)
	if !ok {
		return false
	}
	a := r.newActor(KindCameo, "duong_sinh", "Cụ Gà Dưỡng Sinh", "walk", map[string]string{"color": "xam"}, path[0], now)
	a.Path, a.Speed = path, 1.4
	a.Cues = []*cue{{At: -8.5, Key: "elder.exercise", To: p}}
	p.memo.cameoAt, p.memo.lockouts = now, nil
	return true
}

// npcChoice: người chơi trả lời câu hỏi của cameo (chỉ đường, phỏng vấn, mua của bán rong).
func (r *Room) npcChoice(p *Player, id string, i int, txn txnFunc, fail func(error)) {
	a := r.actors[id]
	if a == nil || a.Target != p || a.State != "ask" {
		fail(errors.New("Câu hỏi đã qua rồi."))
		return
	}
	var choices []string
	for _, c := range a.Cues {
		if len(c.Choice) > 0 {
			choices = c.Choice
		}
	}
	if i < 0 || i >= len(choices) {
		fail(errors.New("Lựa chọn không hợp lệ."))
		return
	}
	a.State = "go"
	now := time.Now()
	switch a.Role {
	case "shipper":
		txn(func() (ActResult, error) {
			if choices[i] != a.Dest {
				r.say(a.ID, a.Name, NPCLines["shipper.lost"][0], a.Pos, p)
				return ActResult{Msg: "Shipper chạy đi hướng đó… hình như sai rồi!"}, nil
			}
			day := DayKey(now)
			if p.acc.ShipTipDay != day {
				p.acc.ShipTipDay, p.acc.ShipTips = day, 0
			}
			r.say(a.ID, a.Name, NPCLines["shipper.thanks"][0], a.Pos, p)
			if p.acc.ShipTips >= ShipperTipsPerDay {
				return ActResult{Msg: "Chỉ đường đúng rồi!"}, nil
			}
			p.acc.ShipTips++
			_ = p.acc.addThoc(1, "npc_tip", "shipper", now)
			return ActResult{Msg: "Chỉ đường đúng: +1 thóc"}, nil
		})
	case "nha_bao":
		r.say(p.PID, p.acc.Char.Name, choices[i], p.pos, nil)
		r.say(a.ID, a.Name, NPCLines["reporter.thanks"][0], a.Pos, p)
		var lootID string
		for j := len(p.acc.Loots) - 1; j >= 0; j-- {
			if p.acc.Loots[j].Worm == a.Dest && !p.acc.Loots[j].Shared {
				lootID = p.acc.Loots[j].ID
				break
			}
		}
		txn(func() (ActResult, error) {
			return ActResult{Msg: "Đã trả lời phỏng vấn!", Data: map[string]any{"loot": lootID}}, nil
		})
	case "ban_rong":
		if i == 2 {
			txn(func() (ActResult, error) { return ActResult{Msg: "Lần sau nhé!"}, nil })
			return
		}
		f := foodByID([]string{"cam", "rau"}[i])
		txn(func() (ActResult, error) {
			if p.acc.Mana >= ManaMax {
				return ActResult{}, errors.New("Bụng đã no căng.")
			}
			if err := p.acc.addThoc(-vendorPrice(f.Cost), "vendor_food", f.ID, now); err != nil {
				return ActResult{}, err
			}
			p.acc.restoreMana(float64(f.Mana))
			p.acc.step("first_meal")
			r.say(a.ID, a.Name, NPCLines["vendor.thanks"][0], a.Pos, p)
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			return ActResult{Msg: fmt.Sprintf("%s: −%d thóc, +%d độ no", f.Name, vendorPrice(f.Cost), f.Mana)}, nil
		})
	default:
		fail(errors.New("Không có gì để trả lời."))
	}
}
