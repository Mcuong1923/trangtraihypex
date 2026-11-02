package main

import (
	"math"
	"strings"
	"testing"
	"time"
)

// Làng sống qua WebSocket thật (PLAN_LOGIC_GAME 5–9): Cụ Cục gọi gà mới, dấu hướng dẫn, điểm danh, quà khởi
// nghiệp, xe buýt vòng làng với lượt miễn phí, xe ôm chở tới khu, trận biểu diễn ở Sân tỷ thí.

func (c *client) expectWhere(typ string, within time.Duration, ok func(msg) bool) msg {
	c.t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		m := c.expect(typ, time.Until(deadline))
		if ok(m) {
			return m
		}
	}
	c.t.Fatalf("%s: không thấy %s phù hợp", c.name, typ)
	return nil
}

func TestNPCWelcomeStarterAndCheckIn(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.StarterThoc, b.JoinGiftsOff = 100, false })
	a, _ := e.playerTok(t, "nga", "Gà Nga")
	// Quà khởi nghiệp 100 + điểm danh ngày 1 (20 thóc, 1 cám) đã vào ví trước khi gửi welcome.
	if a.num("thoc") != 120 || a.count("food", "cam") != 1 {
		t.Fatalf("khởi nghiệp + điểm danh: thóc %v cám %v", a.num("thoc"), a.count("food", "cam"))
	}
	mark := a.expect("mark", 3*time.Second)
	if mark["step"] != "met_elder" || mark["npc"] != "ga_cu_cuc" {
		t.Fatalf("bước đầu tiên phải là gặp Cụ Cục: %v", mark)
	}
	say := a.expectWhere("say", 5*time.Second, func(m msg) bool { return m["id"] == "ga_cu_cuc" })
	if !strings.Contains(say["text"].(string), "Gà Nga") || say["to"] == nil {
		t.Fatalf("Cụ Cục phải gọi đúng tên gà mới: %v", say)
	}
	// Gặp Cụ Cục xong thì dấu chuyển sang bước chạy nhanh.
	a.walkToItem("lobby", "ga_cu_cuc")
	a.mustOK("talk", msg{"id": "ga_cu_cuc"})
	if m := a.expect("mark", 3*time.Second); m["step"] != "first_run" {
		t.Fatalf("bước kế là chạy nhanh: %v", m)
	}
	// Chạy nhanh: gửi input có cờ run thì bước hướng dẫn hoàn thành và nhận thể lực.
	for i := 0; i < 15; i++ {
		a.send(msg{"t": "in", "x": 0, "z": 1, "run": true})
		time.Sleep(60 * time.Millisecond)
	}
	a.send(msg{"t": "in", "x": 0, "z": 0})
	st := a.expect("stam", 2*time.Second)
	if st["s"].(float64) >= 1 {
		t.Fatalf("chạy phải tiêu thể lực: %v", st)
	}
	if m := a.expect("mark", 3*time.Second); m["step"] != "reach_garden" {
		t.Fatalf("chạy xong thì bước kế là tới Vườn giun: %v", m)
	}
	// Đăng nhập lại cùng ngày: không điểm danh lần hai, không nhận quà khởi nghiệp lần hai.
	thoc := a.num("thoc")
	a.conn.Close()
	time.Sleep(200 * time.Millisecond)
	tok, _ := e.auth(t, "/api/login", "nga", "matkhau1")
	b := e.dial(t, "nga", tok)
	b.expect("welcome", 3*time.Second)
	if b.num("thoc") != thoc {
		t.Fatalf("vào lại cùng ngày không nhận thêm: %v → %v", thoc, b.num("thoc"))
	}
}

func TestVillageBusTokensThenFare(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "xebuyt", "Gà Đi Xe", func(acc *Account) { acc.Thoc = 20 })
	rn := Regions[RegionHypex].Roads
	stop := func(id string) RoadStop {
		for _, s := range rn.Stops {
			if s.ID == id {
				return s
			}
		}
		t.Fatalf("không có bến %s", id)
		return RoadStop{}
	}
	ride := func(from, to string) msg {
		st := stop(from)
		a.walkTo(st.Pos.X, st.Pos.Z)
		for i := 0; i < 120; i++ {
			r := a.mustOK("bus_stops", nil)["data"].(map[string]any)
			here := int(r["here"].(float64))
			if here < 0 {
				t.Fatalf("đứng ở bến %s mà máy chủ không nhận", from)
			}
			if eta := r["stops"].([]any)[here].(map[string]any)["eta"].(float64); eta == 0 {
				break
			}
			time.Sleep(250 * time.Millisecond)
		}
		res := a.mustOK("bus_board", msg{"to": to})
		a.expectWhere("pcar", 3*time.Second, func(m msg) bool { return m["carrier"] != "" })
		a.expectWhere("pcar", 60*time.Second, func(m msg) bool { return m["carrier"] == "" })
		if d := a.position().Dist(stop(to).Pos); d > 3 {
			t.Fatalf("xuống xe phải ở bến %s, cách %.1f m", to, d)
		}
		return res
	}
	e.warp(a, "village")
	thoc := a.num("thoc")
	if r := ride("lobby", "garden"); r["data"].(map[string]any)["token"] != true {
		t.Fatal("chuyến 1 dùng lượt miễn phí")
	}
	if r := ride("garden", "market"); r["data"].(map[string]any)["token"] != true {
		t.Fatal("chuyến 2 dùng lượt miễn phí")
	}
	r := ride("market", "lobby")
	if r["data"].(map[string]any)["token"] == true || a.num("thoc") != thoc-VillageFare {
		t.Fatalf("hết 2 lượt thì trả %d thóc: %v → %v", VillageFare, thoc, a.num("thoc"))
	}
	if a.num("busTokens") != 0 || a.num("nextTokenIn") <= 0 {
		t.Fatal("hết lượt phải có đồng hồ hồi lượt")
	}
}

func TestXeOmCarriesForOneOrTwoThoc(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "xeom", "Gà Gọi Xe", func(acc *Account) { acc.Thoc = 20 })
	e.warp(a, "village")
	hub := Regions[RegionHypex].Hub
	a.walkTo(hub.X-4, hub.Z+12) // cạnh bến xe ôm cây đa
	thoc := a.num("thoc")
	a.mustOK("xeom_call", nil)
	offer := a.expect("offer", 10*time.Second)
	if offer["kind"] != "xeom" {
		t.Fatalf("phải nhận lời mời xe ôm: %v", offer)
	}
	route, length, _ := Regions[RegionHypex].Roads.Route(a.position(), "market")
	want := xeomFare(newAcc(), length, time.Now()) // 1–2 thóc theo quãng đường (+1 khi trời mưa)
	_ = route
	r := a.mustOK("xeom_accept", msg{"id": offer["actor"], "to": "market"})
	if fare := r["data"].(map[string]any)["fare"].(float64); int(fare) != want || a.num("thoc") != thoc-float64(want) {
		t.Fatalf("giá xe ôm %v, muốn %d (%.0f m)", fare, want, length)
	}
	a.expectWhere("pcar", 40*time.Second, func(m msg) bool { return m["carrier"] == "" })
	entry := *Zones["market"].Entry
	if d := a.position().Dist(entry); d > 6 {
		t.Fatalf("xe ôm phải trả khách ở cổng chợ, cách %.1f m", d)
	}
	if math.IsNaN(length) || length <= 0 {
		t.Fatal("độ dài tuyến")
	}
}

func TestArenaShowStartsWhenSomeoneWatches(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.playerTok(t, "xem", "Gà Xem Đấu")
	e.warp(a, "arena")
	sc := a.expect("scene", 5*time.Second)
	if sc["kind"] != "arena_show" || len(sc["cues"].([]any)) < 5 {
		t.Fatalf("trận biểu diễn phải có kịch bản: %v", sc)
	}
	names := sc["names"].([]any)
	if names[0] == names[1] {
		t.Fatal("hai đấu sĩ khác nhau")
	}
}

// "Tôi biết chơi rồi" (PLAN_LOGIC_GAME 6.1): bỏ qua vẫn giữ quà khởi nghiệp, dấu hướng dẫn tắt, không nhận
// thưởng hướng dẫn nữa.
func TestTutorialSkipKeepsStarter(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.StarterThoc, b.JoinGiftsOff = 100, false })
	a, _ := e.playerTok(t, "boqua", "Gà Bỏ Qua")
	if m := a.expect("mark", 3*time.Second); m["step"] != "met_elder" {
		t.Fatalf("bước đầu: %v", m)
	}
	thoc := a.num("thoc")
	if thoc < 100 {
		t.Fatalf("quà khởi nghiệp: %v", thoc)
	}
	a.mustOK("tutorial_skip", nil)
	if m := a.expectWhere("mark", 3*time.Second, func(m msg) bool { return m["step"] == "" }); m["npc"] != "" {
		t.Fatalf("bỏ qua thì tắt dấu hướng dẫn: %v", m)
	}
	me := a.mustOK("title", msg{"id": ""})["me"].(map[string]any)
	if me["thoc"].(float64) != thoc || me["tutorialClaimed"] != true {
		t.Fatalf("bỏ qua giữ nguyên thóc, không thưởng: %v → %v", thoc, me["thoc"])
	}
	a.walkToItem("lobby", "ga_cu_cuc")
	if r := a.act("claim_tutorial", nil); r["ok"] == true {
		t.Fatal("đã bỏ qua thì không nhận thưởng hướng dẫn")
	}
	if r := a.act("tutorial_skip", nil); r["ok"] == true {
		t.Fatal("bỏ qua một lần là đủ")
	}
}
