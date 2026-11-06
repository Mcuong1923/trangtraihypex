package main

import (
	"testing"
	"time"
)

// fillRoom giữ chỗ giả cho tới khi chuồng đủ RoomCapacity (không cần 50 tài khoản thật).
func fillRoom(r *Room) {
	for r.Count() < RoomCapacity {
		r.reserve()
	}
}

// Chọn chuồng đầy thì vào hàng chờ; có người rời thì tự được chuyển sang.
func TestRoomQueueMovesWhenSlotFrees(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "chokenh", "Gà Chờ Kênh", func(*Account) {})
	first := e.roomOf(t)
	a.send(msg{"t": "switch_room", "room": "new"})
	a.expect("welcome", 2*time.Second)
	fillRoom(first)
	a.send(msg{"t": "switch_room", "room": first.ID})
	a.toastHas("Bạn đứng thứ 1 trong hàng chờ")
	a.send(msg{"t": "rooms"})
	for _, r := range a.expect("rooms", 2*time.Second)["rooms"].([]any) {
		if ri := r.(map[string]any); ri["id"] == first.ID && (ri["mine"] != 1.0 || ri["queue"] != 1.0) {
			t.Fatalf("danh sách chuồng phải hiện hàng chờ: %v", ri)
		}
	}
	first.unreserve() // một người rời chuồng
	w := a.expect("welcome", 4*time.Second)
	if id := w["room"].(map[string]any)["id"]; id != first.ID {
		t.Fatalf("tới lượt phải được chuyển sang %s, đang ở %v", first.ID, id)
	}
	e.hub.mu.Lock()
	n := len(e.hub.roomQ)
	e.hub.mu.Unlock()
	if n != 0 {
		t.Fatal("chuyển xong phải rời hàng chờ")
	}
}

// Huỷ chờ thì rời hàng.
func TestRoomQueueCancel(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "huycho", "Gà Huỷ Chờ", func(*Account) {})
	first := e.roomOf(t)
	a.send(msg{"t": "switch_room", "room": "new"})
	a.expect("welcome", 2*time.Second)
	fillRoom(first)
	a.send(msg{"t": "switch_room", "room": first.ID})
	a.toastHas("hàng chờ")
	a.send(msg{"t": "room_queue_cancel"})
	a.toastHas("Đã rời hàng chờ")
	e.hub.mu.Lock()
	n := len(e.hub.roomQ[first.ID])
	e.hub.mu.Unlock()
	if n != 0 {
		t.Fatal("huỷ chờ phải rời hàng")
	}
}

// Chỗ trống dành cho người đang chờ: người vào game không được xếp vào chuồng đó.
func TestRoomQueueBlocksNewcomers(t *testing.T) {
	e := newEnv(t, t.TempDir())
	e.seeded(t, "chuongdau", "Gà Chuồng Đầu", func(*Account) {})
	first := e.roomOf(t)
	for first.Count() < RoomCapacity-1 {
		first.reserve()
	}
	e.hub.mu.Lock()
	e.hub.roomQ = map[string][]*roomWait{first.ID: {{s: &Session{}, at: time.Now()}}} // giữ h.mu: vòng xét hàng không thấy
	picked := e.hub.pickRoomLocked(first.ID, first.Region, 1)
	e.hub.roomQ = nil
	e.hub.mu.Unlock()
	if picked == first {
		t.Fatal("chuồng còn 1 chỗ nhưng có người chờ thì người mới không được vào")
	}
}

// Tới lượt mà đang trên xe: không mất chỗ, vẫn đứng đầu hàng; xuống xe thì được chuyển.
func TestRoomQueueBusyKeepsHeadSpot(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, aid := e.seeded(t, "dangban", "Gà Đang Bận", func(*Account) {})
	first := e.roomOf(t)
	a.send(msg{"t": "switch_room", "room": "new"})
	a.expect("welcome", 2*time.Second)
	fillRoom(first)
	a.send(msg{"t": "switch_room", "room": first.ID})
	a.toastHas("hàng chờ")
	e.hub.mu.Lock()
	s := e.hub.sessions[aid]
	e.hub.mu.Unlock()
	cur := s.room.Load()
	setRide := func(on bool) {
		cur.DoWait(func() {
			p := cur.players[s]
			if on {
				p.ride = &Ride{Kind: "xe_dap", Seats: []*Player{p}}
			} else {
				p.ride = nil
			}
		})
	}
	setRide(true)
	first.unreserve()
	a.toastHas("Bạn vẫn đứng đầu hàng")
	if n := first.Count(); n != RoomCapacity-1 {
		t.Fatalf("lượt bận phải trả chỗ giữ, count=%d", n)
	}
	e.hub.mu.Lock()
	head := e.hub.queueAheadLocked(first.ID, s)
	e.hub.mu.Unlock()
	if head != 0 {
		t.Fatal("đang bận vẫn phải đứng đầu hàng")
	}
	setRide(false)
	if id := a.expect("welcome", 6*time.Second)["room"].(map[string]any)["id"]; id != first.ID {
		t.Fatalf("xong việc phải được chuyển sang %s, đang ở %v", first.ID, id)
	}
}

// Đã tự đổi sang chuồng khác trước khi lượt chờ kịp chạy: lượt chờ không kéo người chơi đi nữa (trả chỗ).
func TestRoomQueueGrantSkipsWhenPlayerMovedAway(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, aid := e.seeded(t, "doichokhac", "Gà Đổi Chỗ Khác", func(*Account) {})
	e.hub.mu.Lock()
	s := e.hub.sessions[aid]
	e.hub.mu.Unlock()
	queuedFrom := s.room.Load()                    // lúc vào hàng đang ở chuồng đầu
	a.send(msg{"t": "switch_room", "room": "new"}) // rồi tự đổi sang chuồng mới
	a.expect("welcome", 2*time.Second)
	now := s.room.Load()
	if now == queuedFrom {
		t.Fatal("phải đã sang chuồng mới")
	}
	e.hub.mu.Lock()
	target := e.hub.newRoomLocked(RegionHypex) // chuồng đang chờ
	e.hub.mu.Unlock()
	target.reserve() // vòng xét hàng vừa giữ chỗ
	e.hub.moveQueued(&roomWait{s: s, at: time.Now()}, queuedFrom, target)
	if target.Count() != 0 {
		t.Fatalf("phải trả chỗ đã giữ, count=%d", target.Count())
	}
	if s.room.Load() != now {
		t.Fatal("người chơi đã tự đổi chuồng thì phải ở nguyên chỗ")
	}
}
