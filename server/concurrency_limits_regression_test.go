package main

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test hồi quy cho C5, H1–H5 trong plans/reports/code-reviewer-261105-1412-game-logic-review.md.

// playersOf đếm số Player của một tài khoản trên mọi phòng đang mở (công khai + nhà).
func (e *env) playersOf(id string) int {
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	for _, r := range e.hub.homes {
		rooms = append(rooms, r)
	}
	e.hub.mu.Unlock()
	n := 0
	for _, r := range rooms {
		r.DoWait(func() {
			for _, p := range r.players {
				if p.acc.ID == id {
					n++
				}
			}
		})
	}
	return n
}

// roomCountsConsistent: sĩ số giữ chỗ của mỗi phòng công khai khớp số người thật trong phòng (không có gà "ma").
func (e *env) roomCountsConsistent(t *testing.T) {
	t.Helper()
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	e.hub.mu.Unlock()
	for _, r := range rooms {
		n := 0
		r.DoWait(func() { n = len(r.players) })
		if n != r.Count() {
			t.Fatalf("phòng %s: %d người nhưng sĩ số %d", r.ID, n, r.Count())
		}
	}
}

// C5: hai "hello" cùng token gửi đồng thời không được tạo hai phiên điều khiển cùng một tài khoản.
func TestConcurrentHelloSingleSession(t *testing.T) {
	e := newEnv(t, t.TempDir())
	_, id := e.seeded(t, "haiphien", "Gà Hai Phiên", func(*Account) {})
	tok, _ := e.auth(t, "/api/login", "haiphien", "matkhau1")
	url := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws"
	for round := 0; round < 5; round++ {
		var conns []*websocket.Conn
		for i := 0; i < 2; i++ {
			c, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				t.Fatal(err)
			}
			go func() { // xả tin đến để server không nghẽn gửi
				for {
					if _, _, err := c.ReadMessage(); err != nil {
						return
					}
				}
			}()
			conns = append(conns, c)
		}
		var start, done sync.WaitGroup
		start.Add(1)
		for _, c := range conns {
			done.Add(1)
			go func() {
				defer done.Done()
				start.Wait()
				c.WriteJSON(map[string]any{"t": "hello", "token": tok})
			}()
		}
		start.Done()
		done.Wait()
		// Chờ tới khi có nhân vật (máy chậm), rồi thêm một nhịp để phiên thứ hai (nếu lọt) kịp vào phòng.
		for deadline := time.Now().Add(3 * time.Second); e.playersOf(id) == 0 && time.Now().Before(deadline); {
			time.Sleep(20 * time.Millisecond)
		}
		time.Sleep(200 * time.Millisecond)
		if n := e.playersOf(id); n != 1 {
			t.Fatalf("lượt %d: tài khoản có %d nhân vật đang điều khiển, phải là 1", round, n)
		}
		for _, c := range conns {
			t.Cleanup(func() { c.Close() })
		}
	}
}

// H1: lặp "chuồng mới" không được đẻ phòng vô hạn.
func TestSwitchNewRoomReusesEmpty(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "chuongmoi", "Gà Chuồng Mới", func(*Account) {})
	for i := 0; i < 6; i++ {
		a.send(msg{"t": "switch_room", "room": "new"})
		a.expect("welcome", 2*time.Second)
	}
	e.hub.mu.Lock()
	n := 0
	for _, r := range e.hub.rooms {
		if r.Region == RegionHypex {
			n++
		}
	}
	e.hub.mu.Unlock()
	if n > 2 {
		t.Fatalf("1 người bấm \"chuồng mới\" 6 lần tạo %d phòng; tối đa là số phòng có người + 1", n)
	}
	e.roomCountsConsistent(t)
}

// H2: câu cá có trần mỗi ngày.
func TestFishingDailyCap(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "cauhet", "Gà Câu Hết", func(acc *Account) {
		acc.Mana = 50
		acc.EnsureDaily(time.Now())
		acc.DailyFish = e.hub.bal.FishDailyLimit
	})
	if e.hub.bal.FishDailyLimit != FishDailyDefault {
		t.Fatalf("bộ cân bằng thiếu trần câu cá: %d", e.hub.bal.FishDailyLimit)
	}
	e.warp(a, "park")
	a.walkToItem("park", "cau_ca")
	r := a.act("fish_start", msg{"id": "cau_ca"})
	if r["ok"] == true || !strings.Contains(r["err"].(string), "đủ") {
		t.Fatalf("đủ trần cá trong ngày phải bị chặn: %v", r["err"])
	}
}

// H3: hai lần chuyển vùng chạy song song cho cùng một phiên (bấm đúp "về nhà") không tạo gà "ma".
func TestConcurrentTransferNoGhost(t *testing.T) {
	e := newEnv(t, t.TempDir())
	_, id := e.seeded(t, "bamdup", "Gà Bấm Đúp", func(*Account) {})
	e.hub.mu.Lock()
	s := e.hub.sessions[id]
	e.hub.mu.Unlock()
	for round := 0; round < 5; round++ {
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				e.hub.TransferRegion([]*Session{s}, RegionHypex, []*arrival{{Zone: ZoneLobby}})
			}()
		}
		wg.Wait()
		if n := e.playersOf(id); n != 1 {
			t.Fatalf("lượt %d: phiên có mặt %d lần, phải là 1", round, n)
		}
		e.roomCountsConsistent(t)
	}
}

// H4: quà chỉ đường cho shipper lưu theo tài khoản (đổi kênh không đếm lại từ 0).
func TestShipperTipsStoredOnAccount(t *testing.T) {
	a := &Account{ShipTipDay: DayKey(time.Now()), ShipTips: ShipperTipsPerDay}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	b := &Account{}
	if err := json.Unmarshal(raw, b); err != nil {
		t.Fatal(err)
	}
	if b.ShipTipDay != a.ShipTipDay || b.ShipTips != ShipperTipsPerDay {
		t.Fatal("bộ đếm quà shipper phải được lưu cùng tài khoản")
	}
}

// H5: xô rò giới hạn tần suất message; act vượt giới hạn vẫn nhận "res" lỗi.
func TestMessageRateLimit(t *testing.T) {
	s := &Session{}
	now := time.Now()
	for i := 0; i < int(MsgBurst); i++ {
		if s.rateMsg(now) != msgAllow {
			t.Fatalf("message thứ %d trong giới hạn phải qua", i+1)
		}
	}
	for i := 0; i < int(MsgBurst); i++ {
		if s.rateMsg(now) != msgThrottle {
			t.Fatalf("vượt MsgBurst lần %d: act phải nhận lỗi", i+1)
		}
	}
	if s.rateMsg(now) != msgDrop {
		t.Fatal("vượt 2×MsgBurst phải bỏ hẳn, không đẩy vào phòng")
	}
	if s.rateMsg(now.Add(5*time.Second)) != msgAllow {
		t.Fatal("dừng spam vài giây thì xô rò phải cho qua lại")
	}

	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "spam", "Gà Spam", func(*Account) {})
	const n = 80
	for i := 0; i < n; i++ {
		a.send(msg{"t": "act", "rid": "spam-" + string(rune('A'+i%26)) + string(rune('a'+i/26)), "a": "khong_co_lenh_nay"})
	}
	got, throttled := 0, 0
	deadline := time.After(5 * time.Second)
	for got < n {
		select {
		case m := <-a.in:
			if m["t"] != "res" {
				continue
			}
			got++
			if err, _ := m["err"].(string); strings.Contains(err, "nhanh quá") {
				throttled++
			}
		case <-deadline:
			t.Fatalf("chỉ nhận %d/%d phản hồi: act bị chặn vẫn phải có res", got, n)
		}
	}
	if throttled == 0 {
		t.Fatal("spam act phải bị giới hạn tần suất")
	}
}

// H5: mỗi người tối đa ReportsPerDay báo cáo đang mở; cắt hàng đợi bỏ báo cáo đã xử lý trước.
func TestReportFloodLimits(t *testing.T) {
	so, err := OpenSocial(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	me, target := randHex(12), randHex(12)
	for i := 0; i < ReportsPerDay; i++ {
		if err := so.Report(Report{At: time.Now(), Reporter: me, Target: target, Reason: "spam"}); err != nil {
			t.Fatalf("báo cáo thứ %d: %v", i+1, err)
		}
	}
	if err := so.Report(Report{At: time.Now(), Reporter: me, Target: target, Reason: "spam"}); err == nil {
		t.Fatal("vượt số báo cáo mỗi ngày phải bị chặn")
	}
	rs := []Report{{ID: "o1", Status: "open"}, {ID: "c1", Status: "dismissed"}, {ID: "o2", Status: "open"}, {ID: "c2", Status: "actioned"}, {ID: "o3", Status: "open"}}
	out := trimReports(rs, 3)
	if len(out) != 3 || out[0].ID != "o1" || out[1].ID != "o2" || out[2].ID != "o3" {
		t.Fatalf("cắt hàng đợi phải giữ báo cáo đang mở: %+v", out)
	}
}
