package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Kiểm thử end-to-end qua WebSocket thật (mục 20): đăng ký → tạo gà → sảnh chung → đào → bán/ăn →
// idempotency → nhiều người trong chuồng → PK → phiên mới thay phiên cũ, tài sản giữ nguyên.

func testBalance() Balance {
	return finalize(Balance{Name: "test", DigCooldown: 300 * time.Millisecond, DigDailyLimit: 120, DigMinDuration: 50 * time.Millisecond,
		ManaDrainPerSec: 0.01, ManaPerDig: 1, HungerSlowSec: 300, HungerExhaustSec: 600,
		RescueDuration: time.Second, RescueReward: 5, TutorialReward: 100, ShowcaseEvery: 60 * time.Second,
		BuildDuration: time.Second, StarterThoc: -1, TugSpeed: 10, BusLoopSpeed: 45, BusLoopDwell: 2 * time.Second, JoinGiftsOff: true}) // quà khởi nghiệp có test riêng; tắt để các test đếm thóc giữ nguyên
}

type env struct {
	srv   *httptest.Server
	hub   *Hub
	store *Store
}

func newEnv(t *testing.T, dir string) *env { return newEnvBal(t, dir, nil) }

func newEnvBal(t *testing.T, dir string, tune func(*Balance)) *env {
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	bal := testBalance()
	if tune != nil {
		tune(&bal)
	}
	hub := NewHub(store, bal)
	e := &env{srv: httptest.NewServer(hub.Routes()), hub: hub, store: store}
	// Chạy trước khi xoá thư mục tạm: đợi các phòng lưu xong tài khoản của người vừa thoát.
	t.Cleanup(func() { e.srv.Close(); time.Sleep(300 * time.Millisecond) })
	return e
}

func (e *env) auth(t *testing.T, path, user, pass string) (string, string) {
	b, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	r, err := http.Post(e.srv.URL+path, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out map[string]string
	json.NewDecoder(r.Body).Decode(&out)
	return out["token"], out["err"]
}

type msg map[string]any

type client struct {
	t    *testing.T
	name string
	conn *websocket.Conn
	in   chan msg

	mu      sync.Mutex
	pid     string
	pos     Vec
	zone    string
	me      msg
	seq     int
	backlog []msg // tin đến trong lúc chờ phản hồi hành động, giữ lại cho expect
}

func (e *env) dial(t *testing.T, name, token string) *client {
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, name: name, conn: conn, in: make(chan msg, 4096)}
	go c.read()
	c.send(msg{"t": "hello", "token": token})
	t.Cleanup(func() { conn.Close() })
	return c
}

func (c *client) read() {
	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			close(c.in)
			return
		}
		var m msg
		json.Unmarshal(b, &m)
		c.mu.Lock()
		switch m["t"] {
		case "welcome", "zone":
			if m["t"] == "welcome" {
				c.pid = m["pid"].(string)
				c.me = m["me"].(map[string]any)
			}
			c.zone = m["zone"].(string)
			p := m["pos"].(map[string]any)
			c.pos = Vec{p["x"].(float64), p["z"].(float64)}
		case "snap":
			for i, pid := range m["pids"].([]any) {
				if pid == c.pid {
					row := m["p"].([]any)[i].([]any)
					c.pos = Vec{row[0].(float64), row[1].(float64)}
				}
			}
		case "res":
			c.me = m["me"].(map[string]any)
		}
		c.mu.Unlock()
		if m["t"] != "snap" && m["t"] != "vit" {
			c.in <- m
		}
	}
}

func (c *client) send(m msg) {
	if err := c.conn.WriteJSON(m); err != nil {
		c.t.Fatalf("%s gửi: %v", c.name, err)
	}
}

func (c *client) expect(typ string, within time.Duration) msg {
	c.t.Helper()
	for i, m := range c.backlog {
		if m["t"] == typ {
			c.backlog = append(c.backlog[:i], c.backlog[i+1:]...)
			return m
		}
	}
	deadline := time.After(within)
	for {
		select {
		case m, ok := <-c.in:
			if !ok {
				c.t.Fatalf("%s: kết nối đóng khi chờ %s", c.name, typ)
			}
			if m["t"] == typ {
				return m
			}
		case <-deadline:
			c.t.Fatalf("%s: không nhận được %s", c.name, typ)
		}
	}
}

func (c *client) actRID(rid, a string, args msg) msg {
	c.t.Helper()
	m := msg{"t": "act", "rid": rid, "a": a}
	for k, v := range args {
		m[k] = v
	}
	c.send(m)
	deadline := time.After(5 * time.Second)
	for {
		select {
		case r := <-c.in:
			if r["t"] == "res" && r["rid"] == rid {
				return r
			}
			c.backlog = append(c.backlog, r)
		case <-deadline:
			c.t.Fatalf("%s: %s không có phản hồi", c.name, a)
		}
	}
}

func (c *client) act(a string, args msg) msg {
	c.seq++
	return c.actRID(fmt.Sprintf("%s-%d", c.name, c.seq), a, args)
}

func (c *client) mustOK(a string, args msg) msg {
	c.t.Helper()
	r := c.act(a, args)
	if r["ok"] != true {
		c.t.Fatalf("%s %s: %v", c.name, a, r["err"])
	}
	return r
}

func (c *client) position() Vec {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pos
}

func (c *client) num(key string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.me[key].(float64)
}

func (c *client) count(field, id string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, _ := c.me[field].(map[string]any)[id].(float64)
	return v
}

// walkTo đi theo lộ trình tránh vật cản; vị trí lấy từ snapshot của máy chủ.
func (c *client) walkTo(x, z float64) {
	c.t.Helper()
	c.mu.Lock()
	zone := zoneByID(c.zone)
	if reg := regionOfZone(c.zone); reg != nil {
		zone = reg.World
	}
	c.mu.Unlock()
	for _, wp := range gridRoute(zone, c.position(), Vec{x, z}) {
		c.walkStraight(wp.X, wp.Z)
	}
}

func (c *client) walkStraight(x, z float64) {
	c.t.Helper()
	deadline := time.Now().Add(time.Duration(4+c.position().Dist(Vec{x, z})/WalkSpeed*2) * time.Second)
	for time.Now().Before(deadline) {
		p := c.position()
		dx, dz := x-p.X, z-p.Z
		d := math.Hypot(dx, dz)
		if d < 0.6 {
			c.send(msg{"t": "in", "x": 0, "z": 0})
			time.Sleep(120 * time.Millisecond)
			return
		}
		c.send(msg{"t": "in", "x": dx / d, "z": dz / d})
		time.Sleep(50 * time.Millisecond)
	}
	c.t.Fatalf("%s không đi tới (%.1f, %.1f), đang ở %+v", c.name, x, z, c.position())
}

func regionOfZone(id string) *Region {
	for _, r := range Regions {
		if r.Zones[id] != nil {
			return r
		}
	}
	return nil
}

func zoneByID(id string) *Zone {
	for _, r := range Regions {
		if z := r.Zones[id]; z != nil {
			return z
		}
	}
	return nil
}

// gridRoute: BFS trên lưới 0,5 đơn vị qua vùng đi được, rút gọn thành các đoạn thẳng thông suốt.
func gridRoute(z *Zone, from, to Vec) []Vec {
	clear := func(a, b Vec) bool {
		n := int(math.Ceil(a.Dist(b)/0.2)) + 1
		for i := 1; i <= n; i++ {
			if !z.Walkable(Vec{a.X + (b.X-a.X)*float64(i)/float64(n), a.Z + (b.Z-a.Z)*float64(i)/float64(n)}) {
				return false
			}
		}
		return true
	}
	if z == nil || clear(from, to) {
		return []Vec{to}
	}
	const step = 0.5
	type cell struct{ x, z int }
	at := func(c cell) Vec { return Vec{float64(c.x) * step, float64(c.z) * step} }
	start := cell{int(math.Round(from.X / step)), int(math.Round(from.Z / step))}
	prev := map[cell]cell{start: start}
	q := []cell{start}
	var goal *cell
	for len(q) > 0 && goal == nil {
		cur := q[0]
		q = q[1:]
		for _, d := range []cell{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := cell{cur.x + d.x, cur.z + d.z}
			if _, seen := prev[n]; seen || !z.Walkable(at(n)) {
				continue
			}
			prev[n] = cur
			if at(n).Dist(to) < step*1.5 && clear(at(n), to) {
				goal = &n
				break
			}
			q = append(q, n)
		}
	}
	if goal == nil {
		return []Vec{to}
	}
	pts := []Vec{to}
	for c := *goal; c != start; c = prev[c] {
		pts = append([]Vec{at(c)}, pts...)
	}
	out, cur := []Vec{}, from
	for i := 0; i < len(pts); {
		j := i
		for j+1 < len(pts) && clear(cur, pts[j+1]) {
			j++
		}
		out = append(out, pts[j])
		cur = pts[j]
		i = j + 1
	}
	return out
}

// walkToItem đứng ở điểm đi được gần nhất trong tầm tương tác của vật.
func (c *client) walkToItem(zone, id string) {
	c.t.Helper()
	it := zoneByID(zone).Item(id)
	if it == nil {
		c.t.Fatalf("không có %s ở %s", id, zone)
	}
	z := regionOfZone(zone).World
	from, best, bestD := c.position(), Vec{it.X, it.Z + 0.8}, math.Inf(1)
	for r := 0.8; r < math.Max(1, it.R-0.2); r += 0.4 {
		for a := 0.0; a < 2*math.Pi; a += math.Pi / 12 {
			p := Vec{it.X + math.Sin(a)*r, it.Z + math.Cos(a)*r}
			if d := p.Dist(from) + r*2; z.Walkable(p) && d < bestD {
				best, bestD = p, d
			}
		}
	}
	c.walkTo(best.X, best.Z)
}

// goZone đi bộ thật trên bản đồ liền: ra đầu đường của khu đang đứng → quảng trường → đường vào khu đích.
// Không có cổng dịch chuyển; khu được máy chủ tính theo vị trí.
func (c *client) goZone(_, to string) {
	c.t.Helper()
	c.mu.Lock()
	from := c.zone
	c.mu.Unlock()
	reg := regionOfZone(from)
	if reg == nil || reg.Zones[to] == nil {
		c.t.Fatalf("%s: %s và %s không cùng một trại", c.name, from, to)
	}
	ring := func(e *Vec) Vec {
		l := math.Hypot(e.X-reg.Hub.X, e.Z-reg.Hub.Z)
		return Vec{reg.Hub.X + (e.X-reg.Hub.X)/l*8, reg.Hub.Z + (e.Z-reg.Hub.Z)/l*8}
	}
	var stops []Vec
	if e := reg.Zones[from].Entry; e != nil && from != to {
		stops = append(stops, *e, ring(e))
	}
	dst := reg.Zones[to]
	if dst.Entry != nil {
		stops = append(stops, ring(dst.Entry), *dst.Entry)
	}
	stops = append(stops, dst.Spawns[0])
	for _, st := range stops {
		c.walkTo(st.X, st.Z)
	}
	if got := reg.AreaAt(c.position()); got != to {
		c.t.Fatalf("%s: đi tới %s nhưng đang ở %s %+v", c.name, to, got, c.position())
	}
	c.mu.Lock()
	c.zone = to
	c.mu.Unlock()
}

func (e *env) pidOf(c *client) string {
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	e.hub.mu.Unlock()
	pid := ""
	for _, r := range rooms {
		r.DoWait(func() {
			for _, p := range r.players {
				if p.acc.Username == c.name {
					pid = p.PID
				}
			}
		})
	}
	return pid
}

// areaOf đọc khu hiện tại của người chơi từ máy chủ (khu đổi theo vị trí, không có tin "zone").
func (e *env) areaOf(c *client) string {
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	e.hub.mu.Unlock()
	area := ""
	for _, r := range rooms {
		r.DoWait(func() {
			for _, p := range r.players {
				if p.acc.Username == c.name {
					area = p.zone
				}
			}
		})
	}
	return area
}

// warp chỉ dùng để dựng bối cảnh cho test không kiểm tra việc đi đường (giảm thời gian chạy).
func (e *env) warp(c *client, to string) {
	c.t.Helper()
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	e.hub.mu.Unlock()
	for _, r := range rooms {
		r.DoWait(func() {
			for _, p := range r.players {
				if p.acc.Username == c.name {
					r.moveZone(p, to)
				}
			}
		})
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		z := c.zone
		c.mu.Unlock()
		if z == to {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	c.t.Fatalf("%s: không chuyển được tới %s", c.name, to)
}

func (c *client) dig(spot string) msg {
	c.t.Helper()
	r := c.mustOK("dig_start", msg{"id": spot})
	out := c.mustOK("dig_finish", c.tugPlan(r, true))
	time.Sleep(320 * time.Millisecond) // chờ hồi đất của bộ cân bằng test
	return out
}

// tugPlan chơi giằng co bằng chính mô phỏng của máy chủ (PlanTug, phản xạ tức thì) và chờ đúng thời gian
// thật trước khi gửi; wait=false gửi ngay để thử chặn "xong sớm hơn đồng hồ".
func (c *client) tugPlan(start msg, wait bool) msg {
	c.t.Helper()
	d := start["data"].(map[string]any)
	var p TugParams
	raw, _ := json.Marshal(d["params"])
	json.Unmarshal(raw, &p)
	bursts := TugBursts(uint32(d["seed"].(float64)), p)
	var ev []TugEvent
	var res TugResult
	for _, th := range [][2]float64{{.4, .6}, {.3, .5}, {.45, .65}, {.2, .4}} {
		ev = PlanTug(p, bursts, th[0], th[1], 0)
		if res = SimulateTug(p, bursts, ev); res.Caught {
			break
		}
	}
	if wait {
		wait, _ := d["peckMs"].(float64)
		if bite, ok := d["biteMs"].(float64); ok {
			wait = bite // câu cá: chờ phao chìm thay cho lúc mổ đất
		}
		lead := time.Duration(wait+d["revealMs"].(float64)) * time.Millisecond
		time.Sleep(lead + time.Duration(res.End*float64(time.Second)) + 30*time.Millisecond)
	}
	evs := [][2]float64{}
	for _, e := range ev {
		h := 0.0
		if e.Hold {
			h = 1
		}
		evs = append(evs, [2]float64{e.T, h})
	}
	return msg{"id": d["id"], "ev": evs}
}

func TestEndToEnd(t *testing.T) {
	dir := t.TempDir()
	e := newEnv(t, dir)

	if _, err := e.auth(t, "/api/register", "AB", "123456"); err == "" {
		t.Fatal("tên đăng nhập sai định dạng phải bị từ chối")
	}
	tokA, err := e.auth(t, "/api/register", "alice", "matkhau1")
	if err != "" {
		t.Fatal(err)
	}
	if _, err := e.auth(t, "/api/register", "alice", "khac123"); err == "" {
		t.Fatal("không được đăng ký trùng tên")
	}
	if _, err := e.auth(t, "/api/login", "alice", "saimatkhau"); err == "" {
		t.Fatal("sai mật khẩu phải bị từ chối")
	}

	// --- Tạo gà, xuất hiện ở sảnh chung ---
	a := e.dial(t, "alice", tokA)
	a.expect("need_char", 2*time.Second)
	a.send(msg{"t": "create_char", "name": "A", "color": "vang", "comb": "don", "shirt": "xanh", "sex": "nu"})
	if !strings.Contains(a.expect("char_err", 2*time.Second)["err"].(string), "ký tự") {
		t.Fatal("tên quá ngắn")
	}
	a.send(msg{"t": "create_char", "name": "Gà Bông", "color": "hong", "comb": "don", "shirt": "xanh", "sex": "nu"})
	a.expect("char_err", 2*time.Second)
	a.send(msg{"t": "create_char", "name": "Gà Bông", "color": "vang", "comb": "kep", "shirt": "xanh", "sex": "nu"})
	w := a.expect("welcome", 2*time.Second)
	if w["zone"] != ZoneLobby || a.num("thoc") != 0 || a.num("mana") != 100 {
		t.Fatalf("phiên đầu phải ở sảnh, 0 thóc, 100 mana: %v %v", w["zone"], a.me)
	}

	// --- Server quyết định vị trí: input quá lớn bị chuẩn hoá, không thể dịch chuyển tức thời ---
	start := a.position()
	a.send(msg{"t": "in", "x": 500, "z": 0})
	time.Sleep(200 * time.Millisecond)
	a.send(msg{"t": "in", "x": 0, "z": 0})
	time.Sleep(150 * time.Millisecond)
	if moved := a.position().Dist(start); moved > WalkSpeed*0.5 {
		t.Fatalf("di chuyển %.2f trong 0.2s — server phải giới hạn tốc độ", moved)
	}

	// --- NPC cần đứng gần; gặp Gà Cụ Cục ---
	if r := a.act("talk", msg{"id": "bang_tin"}); r["ok"] == true {
		a.walkTo(10, -8) // xa bảng tin
	}
	a.walkTo(10, -8)
	if r := a.act("talk", msg{"id": "ga_cu_cuc"}); r["ok"] == true {
		t.Fatal("đứng xa NPC không được nói chuyện")
	}
	a.walkToItem("lobby", "ga_cu_cuc")
	a.mustOK("talk", msg{"id": "ga_cu_cuc"})
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("chưa xong hướng dẫn không được nhận thưởng")
	}

	// --- Không đào được ở sảnh; ra Vườn giun đào ---
	if r := a.act("dig_start", msg{"id": "dat_a"}); r["ok"] == true {
		t.Fatal("không đào ở sảnh")
	}
	a.goZone("ra_vuon", "garden")
	if r := a.act("go", msg{"to": "arena"}); r["ok"] == true {
		t.Fatal("không đi xuyên khu nếu không đứng ở cổng tương ứng")
	}
	a.walkToItem("garden", "dat_a")
	r0 := a.mustOK("dig_start", msg{"id": "dat_a"})
	if r := a.act("dig_finish", a.tugPlan(r0, false)); r["ok"] == true {
		t.Fatal("xong giằng co sớm hơn đồng hồ máy chủ phải bị từ chối")
	}
	a.mustOK("dig_finish", a.tugPlan(r0, true)) // gửi lại đúng lúc thì được
	time.Sleep(350 * time.Millisecond)
	mana0 := a.num("mana")
	var commonLoot string
	for i := 0; i < 4; i++ {
		res := a.dig("dat_a")
		loot := res["data"].(map[string]any)["loot"].(map[string]any)
		if !wormByID(loot["worm"].(string)).Rare {
			commonLoot = loot["id"].(string)
		}
	}
	// Bắt đầu lại giữa chừng để gieo lại loài giun: lượt cũ tính như giun thoát nên đất chưa hồi, bị chặn.
	a.mustOK("dig_start", msg{"id": "dat_a"})
	if r := a.act("dig_start", msg{"id": "dat_a"}); r["ok"] == true {
		t.Fatal("bắt đầu lại để gieo lại giun phải bị chặn bởi hồi đất")
	}
	if a.num("dailyDigs") != 5 || a.num("thoc") != 20 {
		t.Fatalf("5 lượt đào, nhiệm vụ đào 5 con trả 20 thóc: digs %v thóc %v", a.num("dailyDigs"), a.num("thoc"))
	}
	if a.num("mana") > mana0-4+0.5 {
		t.Fatalf("mỗi lượt đào phải trừ mana: %v → %v", mana0, a.num("mana"))
	}
	if commonLoot != "" {
		if r := a.act("show", msg{"id": commonLoot}); r["ok"] == true {
			t.Fatal("giun thường không được khoe")
		}
	}
	if r := a.act("show", msg{"id": "loot_gia_mao"}); r["ok"] == true {
		t.Fatal("thẻ khoe giả phải bị từ chối")
	}

	// --- Ra chợ: bán phải đứng ở sạp; retry cùng request ID không cộng lần hai ---
	time.Sleep(350 * time.Millisecond)
	a.goZone("ve_sanh", "lobby")
	a.goZone("ra_cho", "market")
	var wormID string
	for _, wm := range Worms {
		if a.count("worms", wm.ID) > 0 {
			wormID = wm.ID
			break
		}
	}
	if r := a.act("sell", msg{"id": wormID}); r["ok"] == true {
		t.Fatal("đứng xa sạp không bán được")
	}
	a.walkToItem("market", "sap_giun")
	before, n0 := a.num("thoc"), a.count("worms", wormID)
	r := a.actRID("sell-1", "sell", msg{"id": wormID})
	if r["ok"] != true {
		t.Fatal(r["err"])
	}
	again := a.actRID("sell-1", "sell", msg{"id": wormID})
	if again["msg"] != r["msg"] || a.count("worms", wormID) != n0-1 {
		t.Fatalf("retry phải trả kết quả cũ, không bán thêm: còn %v", a.count("worms", wormID))
	}
	if got := a.num("thoc") - before; got != float64(SellPrice(nil, wormByID(wormID), time.Now())) {
		t.Fatalf("bán nhận %v thóc", got)
	}
	// Bán thêm 2 con hoàn thành nhiệm vụ bán 3 giun.
	for i := 0; i < 2; i++ {
		for _, wm := range Worms {
			if a.count("worms", wm.ID) > 0 {
				a.mustOK("sell", msg{"id": wm.ID})
				break
			}
		}
	}

	// --- Ăn: chế biến giun và mua thức ăn ---
	for _, wm := range Worms {
		if a.count("worms", wm.ID) > 0 {
			a.mustOK("eat_worm", msg{"id": wm.ID})
			break
		}
	}
	if a.num("xp") == 0 && a.num("level") == 1 {
		t.Fatal("ăn giun phải cộng XP")
	}
	if r := a.act("buy_food", msg{"id": "cam", "eat": false}); r["ok"] == true {
		t.Fatal("đứng xa quầy không mua được")
	}
	a.walkToItem("market", "quay_an")
	a.mustOK("buy_food", msg{"id": "cam", "eat": false})
	if a.count("food", "cam") != 1 {
		t.Fatal("thức ăn mua cất phải vào túi")
	}

	// --- Thưởng hướng dẫn: còn thiếu bước chạy nhanh ---
	a.goZone("ve_sanh", "lobby")
	a.walkToItem("lobby", "ga_cu_cuc")
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("thiếu bước chạy nhanh")
	}

	// --- Người thứ hai vào cùng chuồng, chat, biểu cảm ---
	tokB, _ := e.auth(t, "/api/register", "bob", "matkhau2")
	b := e.dial(t, "bob", tokB)
	b.expect("need_char", 2*time.Second)
	b.send(msg{"t": "create_char", "name": "Gà Mơ", "color": "nau", "comb": "ro", "shirt": "none", "sex": "an"})
	wb := b.expect("welcome", 2*time.Second)
	if wb["room"].(map[string]any)["id"] != w["room"].(map[string]any)["id"] {
		t.Fatal("người mới phải vào chuồng đang có người khi còn chỗ")
	}
	if len(wb["players"].([]any)) != 1 {
		t.Fatal("bob phải thấy alice trong sảnh")
	}
	a.expect("pjoin", 2*time.Second)
	b.send(msg{"t": "chat", "text": "  xin chào cả chuồng!  "})
	c := a.expect("chat", 2*time.Second)
	for c["kind"] != "chat" {
		c = a.expect("chat", 2*time.Second)
	}
	if c["text"] != "xin chào cả chuồng!" || c["name"] != "Gà Mơ" {
		t.Fatalf("chat: %v", c)
	}
	a.send(msg{"t": "emote", "e": "chao"})
	b.expect("emote", 2*time.Second)
	// Chạy nhanh là bước cuối: Cụ Cục tự gửi thưởng hướng dẫn qua Chim Chíp (PLAN_LOGIC_GAME 6), đúng một lần.
	thoc := a.num("thoc")
	for i := 0; i < 6; i++ {
		a.send(msg{"t": "in", "x": 0, "z": 1, "run": true})
		time.Sleep(60 * time.Millisecond)
	}
	a.send(msg{"t": "in", "x": 0, "z": 0})
	a.expectWhere("me", 3*time.Second, func(m msg) bool { return m["me"].(map[string]any)["tutorialClaimed"] == true })
	if me := a.mustOK("title", msg{"id": ""})["me"].(map[string]any); me["thoc"].(float64) != thoc+100 {
		t.Fatalf("thưởng hướng dẫn 100 thóc: %v → %v", thoc, me["thoc"])
	}
	a.walkToItem("lobby", "ga_cu_cuc")
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("thưởng hướng dẫn chỉ một lần")
	}

	// --- PK giữa hai người: mời, nhận, máy chủ đếm lượt mổ có giới hạn tần suất ---
	a.goZone("ra_san", "arena")
	b.goZone("ra_san", "arena")
	bPID := e.pidOf(b) // cùng bản đồ liền nên đã thấy nhau từ sảnh, không có pjoin mới
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "peck"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})
	st := a.expect("pk_start", 2*time.Second)
	b.expect("pk_start", 2*time.Second)
	if r := a.act("equip", msg{"slot": "head", "id": ""}); r["ok"] == true {
		t.Fatal("không đổi đồ trong trận")
	}
	a.expect("pk_go", 5*time.Second)
	// Spam 200 lần trong tích tắc: server chỉ nhận lượt cách nhau ≥ 60ms.
	for i := 0; i < 200; i++ {
		a.send(msg{"t": "act", "a": "pk_tap", "id": st["id"]})
	}
	for i := 0; i < 10; i++ {
		b.send(msg{"t": "act", "a": "pk_tap", "id": st["id"]})
		time.Sleep(70 * time.Millisecond)
	}
	end := a.expect("pk_end", 12*time.Second)
	b.expect("pk_end", 2*time.Second)
	if end["a"].(float64) > 3 || end["b"].(float64) < 8 || end["winner"].(float64) != 1 {
		t.Fatalf("auto-click phải bị chặn; nhịp hợp lệ thắng: %v", end)
	}

	// --- Phiên mới thay phiên cũ, luôn bắt đầu ở sảnh, tài sản giữ nguyên ---
	thocBefore := a.num("thoc")
	tokA2, _ := e.auth(t, "/api/login", "alice", "matkhau1")
	a2 := e.dial(t, "alice2", tokA2)
	a.expect("kicked", 2*time.Second)
	w2 := a2.expect("welcome", 2*time.Second)
	if w2["zone"] != ZoneLobby || a2.num("thoc") != thocBefore {
		t.Fatalf("phiên mới: khu %v thóc %v (trước %v)", w2["zone"], a2.num("thoc"), thocBefore)
	}

	// --- Dữ liệu đã lưu bền vững xuống đĩa ---
	time.Sleep(100 * time.Millisecond)
	s2, err2 := OpenStore(dir)
	if err2 != nil {
		t.Fatal(err2)
	}
	id, _ := e.store.AccountIDForToken(tokA2)
	acc, err3 := s2.Account(id)
	if err3 != nil || float64(acc.Thoc) != thocBefore || acc.Stats.PKPlayed != 1 || !acc.TutorialClaimed {
		t.Fatalf("đọc lại từ đĩa: %v thóc %d pk %d", err3, acc.Thoc, acc.Stats.PKPlayed)
	}
}

func TestRoomCapacityAndSwitch(t *testing.T) {
	e := newEnv(t, t.TempDir())
	var clients []*client
	for i := 0; i < RoomCapacity+1; i++ {
		tok, err := e.auth(t, "/api/register", fmt.Sprintf("ga%02d", i), "matkhau1")
		if err != "" {
			t.Fatal(err)
		}
		c := e.dial(t, fmt.Sprintf("ga%02d", i), tok)
		c.expect("need_char", 2*time.Second)
		c.send(msg{"t": "create_char", "name": fmt.Sprintf("Gà số %d", i), "color": "vang", "comb": "don", "shirt": "none", "sex": "an"})
		c.expect("welcome", 2*time.Second)
		clients = append(clients, c)
	}
	e.hub.mu.Lock()
	rooms := append([]*Room(nil), e.hub.rooms...)
	e.hub.mu.Unlock()
	if len(rooms) != 2 || rooms[0].Count() != RoomCapacity || rooms[1].Count() != 1 {
		t.Fatalf("26 người: cần 2 chuồng 25 + 1, có %d chuồng", len(rooms))
	}
	// Chuồng đầy thì không đổi sang được; chuồng mới thì được.
	last := clients[RoomCapacity]
	last.send(msg{"t": "switch_room", "room": rooms[0].ID})
	if !strings.Contains(last.expect("toast", 2*time.Second)["text"].(string), "đầy") {
		t.Fatal("phải báo chuồng đầy")
	}
	clients[0].send(msg{"t": "switch_room", "room": rooms[1].ID})
	w := clients[0].expect("welcome", 2*time.Second)
	if w["room"].(map[string]any)["id"] != rooms[1].ID || w["zone"] != ZoneLobby {
		t.Fatal("đổi chuồng phải vào sảnh chuồng đích")
	}
	time.Sleep(50 * time.Millisecond)
	if rooms[0].Count() != RoomCapacity-1 || rooms[1].Count() != 2 {
		t.Fatalf("sĩ số sau khi đổi: %d / %d", rooms[0].Count(), rooms[1].Count())
	}
}
