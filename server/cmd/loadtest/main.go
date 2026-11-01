// Load test (mục 18.2, 20): N bot đăng ký thật, tạo gà, rồi chơi theo tỉ lệ hành vi của plan
// — 60% đi lại/tương tác, 15% về nhà đặt đồ, 15% PK với Gà Máy, 10% chat/đứng — qua WebSocket thật.
//
//	go run ./cmd/loadtest -url http://localhost:8080 -n 1000 -dur 3m -ramp 60s
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	base    = flag.String("url", "http://localhost:8080", "địa chỉ máy chủ")
	n       = flag.Int("n", 200, "số bot")
	dur     = flag.Duration("dur", 2*time.Minute, "thời gian giữ tải sau khi đủ bot")
	ramp    = flag.Duration("ramp", 30*time.Second, "thời gian tăng dần số bot")
	prefix  = flag.String("prefix", fmt.Sprintf("lt%d", time.Now().Unix()%100000), "tiền tố tên đăng nhập")
	verbose = flag.Bool("v", false, "in lỗi từng bot")
)

type stats struct {
	mu        sync.Mutex
	rtts      []float64
	connected atomic.Int64
	peak      atomic.Int64
	failed    atomic.Int64
	msgs      atomic.Int64
	acts      atomic.Int64
	actFail   atomic.Int64
	pk        atomic.Int64
	homes     atomic.Int64
	dropped   atomic.Int64
}

var st stats

func main() {
	flag.Parse()
	fmt.Printf("Load test: %d bot, tăng dần %s, giữ tải %s → %s\n", *n, *ramp, *dur, *base)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			time.Sleep(time.Duration(float64(*ramp) * float64(i) / float64(*n)))
			runBot(i, stop)
		}(i)
	}
	tick := time.NewTicker(10 * time.Second)
	end := time.After(*ramp + *dur)
loop:
	for {
		select {
		case <-tick.C:
			m := status()
			fmt.Printf("[%5.0fs] kết nối %d · lỗi %d · server online %v · tick p95 %.2fms max %.2fms · heap %.0fMB · goroutine %v\n",
				time.Since(start).Seconds(), st.connected.Load(), st.failed.Load(), m["online"], num(m, "tickP95ms"), num(m, "tickMaxms"), num(m, "heapMB"), m["goroutines"])
		case <-end:
			break loop
		}
	}
	m := status()
	close(stop)
	wg.Wait()

	st.mu.Lock()
	sort.Float64s(st.rtts)
	r := st.rtts
	st.mu.Unlock()
	pct := func(p float64) float64 {
		if len(r) == 0 {
			return 0
		}
		return r[int(math.Min(float64(len(r)-1), p*float64(len(r))))]
	}
	el := time.Since(start).Seconds()
	fmt.Println("\n===== Kết quả =====")
	fmt.Printf("Bot kết nối đồng thời (đỉnh): %d/%d (lỗi %d)\n", st.peak.Load(), *n, st.failed.Load())
	fmt.Printf("RTT ping (gồm xếp hàng phía server): p50 %.1fms · p95 %.1fms · max %.1fms (%d mẫu)\n", pct(.5), pct(.95), pct(1), len(r))
	fmt.Printf("Tin nhận: %.0f/giây · hành động gửi %d (bị từ chối hợp lệ %d) · trận PK %d · lượt về nhà %d\n",
		float64(st.msgs.Load())/el, st.acts.Load(), st.actFail.Load(), st.pk.Load(), st.homes.Load())
	fmt.Printf("Máy chủ lúc cao điểm: online %v · phòng %v · tick p50 %.2fms p95 %.2fms max %.2fms (ngân sách 50ms/tick) · heap %.0fMB · gói bị bỏ %v\n",
		m["online"], m["rooms"], num(m, "tickP50ms"), num(m, "tickP95ms"), num(m, "tickMaxms"), num(m, "heapMB"), m["sendDrops"])
}

func num(m map[string]any, k string) float64 { f, _ := m[k].(float64); return f }

func status() map[string]any {
	r, err := http.Get(*base + "/api/status")
	if err != nil {
		return map[string]any{}
	}
	defer r.Body.Close()
	var out struct {
		Metrics map[string]any `json:"metrics"`
	}
	json.NewDecoder(r.Body).Decode(&out)
	return out.Metrics
}

type bot struct {
	id   int
	conn *websocket.Conn
	wmu  sync.Mutex
	mu   sync.Mutex
	pid  string
	pos  [2]float64
	zone string
	in   chan map[string]any
	seq  int
}

func (b *bot) send(m map[string]any) error {
	b.wmu.Lock()
	defer b.wmu.Unlock()
	return b.conn.WriteJSON(m)
}

func (b *bot) where() ([2]float64, string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pos, b.zone
}

func runBot(i int, stop chan struct{}) {
	fail := func(err error) {
		st.failed.Add(1)
		if *verbose {
			fmt.Printf("bot %d: %v\n", i, err)
		}
	}
	user := fmt.Sprintf("%s_%d", *prefix, i)
	body, _ := json.Marshal(map[string]string{"username": user, "password": "matkhau1"})
	resp, err := http.Post(*base+"/api/register", "application/json", bytes.NewReader(body))
	if err != nil {
		fail(err)
		return
	}
	var auth map[string]string
	json.NewDecoder(resp.Body).Decode(&auth)
	resp.Body.Close()
	if auth["token"] == "" {
		fail(fmt.Errorf("đăng ký: %s", auth["err"]))
		return
	}
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(*base, "http")+"/ws", nil)
	if err != nil {
		fail(err)
		return
	}
	defer conn.Close()
	b := &bot{id: i, conn: conn, in: make(chan map[string]any, 256)}
	go b.read()
	b.send(map[string]any{"t": "hello", "token": auth["token"]})
	if b.wait("need_char", 10*time.Second) == nil {
		fail(fmt.Errorf("không nhận need_char"))
		return
	}
	b.send(map[string]any{"t": "create_char", "name": fmt.Sprintf("Bot %d", i), "color": []string{"vang", "kem", "cam", "nau"}[i%4], "comb": "don", "shirt": "none", "sex": "an"})
	if b.wait("welcome", 10*time.Second) == nil {
		fail(fmt.Errorf("không vào được sảnh"))
		return
	}
	if c := st.connected.Add(1); c > st.peak.Load() {
		st.peak.Store(c)
	}
	defer st.connected.Add(-1)

	go func() { // đo RTT mỗi 2 giây
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if b.send(map[string]any{"t": "ping", "c": float64(time.Now().UnixNano()) / 1e6}) != nil {
					return
				}
			}
		}
	}()

	r := rand.Float64()
	switch {
	case r < 0.60:
		b.wander(stop)
	case r < 0.75:
		b.home(stop)
	case r < 0.90:
		b.pk(stop)
	default:
		b.chatter(stop)
	}
}

func (b *bot) read() {
	for {
		_, data, err := b.conn.ReadMessage()
		if err != nil {
			close(b.in)
			return
		}
		st.msgs.Add(1)
		var m map[string]any
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		switch m["t"] {
		case "welcome", "zone":
			b.mu.Lock()
			if m["t"] == "welcome" {
				b.pid, _ = m["pid"].(string)
			}
			b.zone, _ = m["zone"].(string)
			if p, ok := m["pos"].(map[string]any); ok {
				b.pos = [2]float64{p["x"].(float64), p["z"].(float64)}
			}
			b.mu.Unlock()
		case "snap":
			pids, _ := m["pids"].([]any)
			for i, pid := range pids {
				if pid == b.pid {
					row := m["p"].([]any)[i].([]any)
					b.mu.Lock()
					b.pos = [2]float64{row[0].(float64), row[1].(float64)}
					b.mu.Unlock()
				}
			}
			continue
		case "pong":
			c, _ := m["c"].(float64)
			st.mu.Lock()
			st.rtts = append(st.rtts, float64(time.Now().UnixNano())/1e6-c)
			st.mu.Unlock()
			continue
		case "vit", "pjoin", "pleave", "chat", "emote", "fx", "room", "pk_score":
			continue
		}
		select {
		case b.in <- m:
		default:
			st.dropped.Add(1)
		}
	}
}

func (b *bot) wait(t string, d time.Duration) map[string]any {
	deadline := time.After(d)
	for {
		select {
		case m, ok := <-b.in:
			if !ok {
				return nil
			}
			if m["t"] == t {
				return m
			}
		case <-deadline:
			return nil
		}
	}
}

func (b *bot) act(a string, args map[string]any) bool {
	b.seq++
	rid := fmt.Sprintf("b%d-%d", b.id, b.seq)
	m := map[string]any{"t": "act", "rid": rid, "a": a}
	for k, v := range args {
		m[k] = v
	}
	st.acts.Add(1)
	if b.send(m) != nil {
		return false
	}
	deadline := time.After(10 * time.Second)
	for {
		select {
		case r, ok := <-b.in:
			if !ok {
				return false
			}
			if r["t"] == "res" && r["rid"] == rid {
				if r["ok"] != true {
					st.actFail.Add(1)
				}
				return r["ok"] == true
			}
		case <-deadline:
			st.actFail.Add(1)
			return false
		}
	}
}

// walkTo đi bằng input như người chơi; vị trí lấy từ snapshot của máy chủ.
func (b *bot) walkTo(x, z float64, stop chan struct{}) bool {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-stop:
			return false
		default:
		}
		p, _ := b.where()
		dx, dz := x-p[0], z-p[1]
		d := math.Hypot(dx, dz)
		if d < 0.7 {
			b.send(map[string]any{"t": "in", "x": 0, "z": 0})
			time.Sleep(150 * time.Millisecond)
			return true
		}
		if b.send(map[string]any{"t": "in", "x": dx / d, "z": dz / d}) != nil {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func sleepOr(d time.Duration, stop chan struct{}) bool {
	select {
	case <-stop:
		return false
	case <-time.After(d):
		return true
	}
}

func (b *bot) wander(stop chan struct{}) {
	for {
		x, z := rand.Float64()*20-10, rand.Float64()*16-8
		if !b.walkTo(x, z, stop) && !sleepOr(time.Second, stop) {
			return
		}
		if rand.Float64() < 0.3 {
			b.send(map[string]any{"t": "emote", "e": []string{"chao", "vui", "mua", "gay"}[rand.IntN(4)]})
		}
		if rand.Float64() < 0.2 {
			b.act("talk", map[string]any{"id": "bang_tin"}) // tương tác NPC/bảng (thường bị từ chối vì đứng xa: vẫn là tải hợp lệ)
		}
		if !sleepOr(time.Duration(500+rand.IntN(2500))*time.Millisecond, stop) {
			return
		}
	}
}

func (b *bot) home(stop chan struct{}) {
	b.send(map[string]any{"t": "visit", "owner": "me"})
	if b.wait("welcome", 10*time.Second) == nil {
		return
	}
	st.homes.Add(1)
	b.act("home_place", map[string]any{"id": "chau_hoa", "x": -2, "z": -1})
	for {
		x, z := rand.Float64()*6-3, rand.Float64()*5-2.5
		b.walkTo(x, z, stop)
		if !sleepOr(time.Duration(1000+rand.IntN(3000))*time.Millisecond, stop) {
			return
		}
	}
}

func (b *bot) pk(stop chan struct{}) {
	if !b.walkTo(8.5, -7.8, stop) || !b.act("go", map[string]any{"to": "arena"}) {
		b.wander(stop)
		return
	}
	time.Sleep(300 * time.Millisecond)
	if !b.walkTo(-5, -2.0, stop) {
		b.wander(stop)
		return
	}
	for {
		if b.act("pk_bot", map[string]any{"mode": "peck"}) {
			st.pk.Add(1)
			mt := b.wait("pk_start", 5*time.Second)
			if mt != nil && b.wait("pk_go", 6*time.Second) != nil {
				end := time.Now().Add(8 * time.Second)
				for time.Now().Before(end) {
					b.send(map[string]any{"t": "act", "a": "pk_tap", "id": mt["id"]})
					time.Sleep(time.Duration(110+rand.IntN(60)) * time.Millisecond)
				}
				b.wait("pk_end", 4*time.Second)
			}
		}
		if !sleepOr(time.Duration(2000+rand.IntN(3000))*time.Millisecond, stop) {
			return
		}
	}
}

func (b *bot) chatter(stop chan struct{}) {
	lines := []string{"chào cả chuồng", "ai đi đào giun không", "hôm nay vui ghê", "áo mới đẹp không", "ra sân tỷ thí nào"}
	for {
		if rand.Float64() < 0.5 {
			b.send(map[string]any{"t": "chat", "text": lines[rand.IntN(len(lines))]})
		}
		if !sleepOr(time.Duration(4000+rand.IntN(4000))*time.Millisecond, stop) {
			return
		}
	}
}
