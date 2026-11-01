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
		BuildDuration: time.Second})
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

// walkTo gửi input hướng tới đích; vị trí lấy từ snapshot của máy chủ.
func (c *client) walkTo(x, z float64) {
	c.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
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

func (c *client) walkToItem(zone, id string) {
	var it *Item
	for _, r := range Regions {
		if z := r.Zones[zone]; z != nil {
			it = z.Item(id)
		}
	}
	c.walkTo(it.X, it.Z+0.8)
}

func (c *client) goZone(gate, to string) {
	c.t.Helper()
	c.walkToItem(c.zone, gate)
	c.mustOK("go", msg{"to": to}) // tin "zone" tới trước "res" nên c.zone đã cập nhật
	c.mu.Lock()
	zone := c.zone
	c.mu.Unlock()
	if zone != to {
		c.t.Fatalf("%s: vẫn ở %s", c.name, zone)
	}
}

func (c *client) dig(spot string) msg {
	c.t.Helper()
	r := c.mustOK("dig_start", msg{"id": spot})
	time.Sleep(80 * time.Millisecond)
	id := r["data"].(map[string]any)["id"]
	out := c.mustOK("dig_stop", msg{"id": id, "elapsed": 0.07})
	time.Sleep(320 * time.Millisecond) // chờ hồi đất của bộ cân bằng test
	return out
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
	time.Sleep(10 * time.Millisecond)
	if r := a.act("dig_stop", msg{"id": r0["data"].(map[string]any)["id"], "elapsed": 0}); r["ok"] == true {
		t.Fatal("thao tác nhanh hơn ngưỡng tối thiểu phải bị từ chối")
	}
	time.Sleep(50 * time.Millisecond)
	mana0 := a.num("mana")
	var commonLoot string
	for i := 0; i < 5; i++ {
		res := a.dig("dat_a")
		loot := res["data"].(map[string]any)["loot"].(map[string]any)
		if !wormByID(loot["worm"].(string)).Rare {
			commonLoot = loot["id"].(string)
		}
	}
	// Đào ngay khi đất chưa hồi.
	a.mustOK("dig_start", msg{"id": "dat_a"})
	time.Sleep(60 * time.Millisecond)
	r1 := a.mustOK("dig_start", msg{"id": "dat_a"}) // lượt mới thay lượt cũ
	time.Sleep(60 * time.Millisecond)
	a.mustOK("dig_stop", msg{"id": r1["data"].(map[string]any)["id"], "elapsed": 0.05})
	if r := a.act("dig_start", msg{"id": "dat_a"}); r["ok"] == true {
		t.Fatal("đất chưa hồi phải chặn")
	}
	if a.num("dailyDigs") != 6 || a.num("thoc") != 20 {
		t.Fatalf("6 lượt đào, nhiệm vụ đào 5 con trả 20 thóc: digs %v thóc %v", a.num("dailyDigs"), a.num("thoc"))
	}
	if a.num("mana") > mana0-6+0.5 {
		t.Fatalf("mỗi lượt đào phải trừ 1 mana: %v → %v", mana0, a.num("mana"))
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
	if got := a.num("thoc") - before; got != float64(wormByID(wormID).Sell) {
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

	// --- Thưởng hướng dẫn: còn thiếu bước xã hội ---
	a.goZone("ve_sanh", "lobby")
	a.walkToItem("lobby", "ga_cu_cuc")
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("thiếu bước xã hội")
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
	// Hoàn thành bước xã hội → nhận thưởng hướng dẫn đúng một lần.
	time.Sleep(100 * time.Millisecond)
	thoc := a.num("thoc")
	a.mustOK("claim_tutorial", nil)
	if a.num("thoc") != thoc+100 {
		t.Fatal("thưởng hướng dẫn 100 thóc")
	}
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("thưởng hướng dẫn chỉ một lần")
	}

	// --- PK giữa hai người: mời, nhận, máy chủ đếm lượt mổ có giới hạn tần suất ---
	a.goZone("ra_san", "arena")
	b.goZone("ra_san", "arena")
	var bPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Mơ" {
			bPID = p["pid"].(string)
			break
		}
	}
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
