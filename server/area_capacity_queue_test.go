package main

import (
	"testing"
	"time"
)

// areaTestRoom: ba gà trong một kênh, vườn chỉ chứa 1 người; a đang đứng trong vườn.
func areaTestRoom(t *testing.T) (*Room, [3]string) {
	t.Helper()
	e := newEnv(t, t.TempDir())
	a, aid := e.seeded(t, "trongvuon", "Gà Trong Vườn", func(*Account) {})
	_, bid := e.seeded(t, "chothu1", "Gà Chờ Một", func(*Account) {})
	_, cid := e.seeded(t, "chothu2", "Gà Chờ Hai", func(*Account) {})
	e.warp(a, "garden")
	old := Zones["garden"].Capacity
	Zones["garden"].Capacity = 1
	t.Cleanup(func() { Zones["garden"].Capacity = old })
	return e.roomOf(t), [3]string{aid, bid, cid}
}

func playersByID(r *Room, ids [3]string) (out [3]*Player) {
	for _, q := range r.players {
		for i, id := range ids {
			if q.acc.ID == id {
				out[i] = q
			}
		}
	}
	return
}

// Khu đầy: người tới trước trong hàng vào trước; chỗ giữ hết hạn thì người sau được vào.
func TestAreaQueueFirstComeFirstIn(t *testing.T) {
	r, ids := areaTestRoom(t)
	r.DoWait(func() {
		p := playersByID(r, ids)
		a, b, c := p[0], p[1], p[2]
		gate := *r.zones["garden"].Entry
		b.pos, c.pos = gate, gate // đứng chờ ở cổng vườn
		if !r.areaFull(b, "garden") || !r.areaFull(c, "garden") {
			t.Fatal("vườn đầy phải chặn cả hai")
		}
		if r.areaQueuePos(b, "garden") != 1 || r.areaQueuePos(c, "garden") != 2 {
			t.Fatalf("thứ tự chờ sai: b=%d c=%d", r.areaQueuePos(b, "garden"), r.areaQueuePos(c, "garden"))
		}
		a.zone = r.world.Outdoor // a ra khỏi vườn: còn 1 chỗ
		if !r.areaFull(c, "garden") {
			t.Fatal("c đứng sau b không được chen vào chỗ trống")
		}
		r.stepAreaQueues(time.Now())
		if w := r.areaQ["garden"][0]; w.p != b || !w.told {
			t.Fatal("người đầu hàng phải được báo tới lượt")
		}
		r.areaQ["garden"][0].at = time.Now().Add(-AreaQueueHold - time.Second) // b không vào kịp
		if r.areaFull(c, "garden") {
			t.Fatal("b bỏ lượt thì c phải vào được")
		}
		if r.areaQueuePos(c, "garden") != 0 {
			t.Fatal("vào được rồi thì phải rời hàng chờ")
		}
	})
}

// Xuống xe ngay trong khu đã đầy: được đặt ra ngoài cổng thay vì lọt vào khu.
func TestDismountIntoFullAreaLandsOutside(t *testing.T) {
	r, ids := areaTestRoom(t)
	r.DoWait(func() {
		p := playersByID(r, ids)
		driver, rider := p[1], p[2]
		in := r.zones["garden"].Spawns[0]
		driver.pos, rider.pos, rider.zone = in, in, "garden" // vườn đã có a (sức chứa 1)
		driver.ride = &Ride{Kind: "xe_dap", Seats: []*Player{driver, rider}}
		rider.rideOf = driver
		r.dismount(rider, "")
		if rider.zone == "garden" || r.world.AreaAt(rider.pos) == "garden" {
			t.Fatalf("khách xuống xe phải ở ngoài vườn đầy, đang ở %s %+v", rider.zone, rider.pos)
		}
		if !r.world.World.Walkable(rider.pos) {
			t.Fatal("chỗ xuống phải đi được")
		}
	})
}

// Còn chỗ thì xuống xe trong khu như bình thường.
func TestDismountIntoAreaWithRoomStaysInside(t *testing.T) {
	r, ids := areaTestRoom(t)
	r.DoWait(func() {
		p := playersByID(r, ids)
		a, driver, rider := p[0], p[1], p[2]
		a.zone = r.world.Outdoor
		in := r.zones["garden"].Spawns[0]
		driver.pos, rider.pos, rider.zone = in, in, "garden"
		driver.ride = &Ride{Kind: "xe_dap", Seats: []*Player{driver, rider}}
		rider.rideOf = driver
		r.dismount(rider, "")
		if rider.zone != "garden" {
			t.Fatalf("vườn còn chỗ thì khách xuống trong vườn, đang ở %s", rider.zone)
		}
	})
}

// Đứng yên ở cổng (không giữ phím) quá AreaQueueHold vẫn giữ chỗ; đi xa cổng thì mất chỗ.
func TestAreaQueueHoldWhileStandingAtGate(t *testing.T) {
	r, ids := areaTestRoom(t)
	r.DoWait(func() {
		p := playersByID(r, ids)
		b, c := p[1], p[2]
		gate := *r.zones["garden"].Entry
		b.pos, c.pos = gate, gate
		r.areaFull(b, "garden")
		r.areaFull(c, "garden")
		r.areaQ["garden"][0].at = time.Now().Add(-AreaQueueHold + time.Second) // b đứng yên gần hết hạn
		r.stepAreaQueues(time.Now())                                           // còn gần cổng: làm mới chỗ
		if r.areaQueuePos(b, "garden") != 1 {
			t.Fatal("đứng yên ở cổng phải còn giữ chỗ")
		}
		b.pos = r.zones[ZoneLobby].Spawns[0] // b bỏ đi chỗ khác
		r.stepAreaQueues(time.Now())
		if r.areaQueuePos(b, "garden") != 0 || r.areaQueuePos(c, "garden") != 1 {
			t.Fatalf("đi xa cổng phải mất chỗ: b=%d c=%d", r.areaQueuePos(b, "garden"), r.areaQueuePos(c, "garden"))
		}
	})
}

// Khách còn ngồi trên xe khi vào kênh/đi qua khu đầy thì không bị dời (khách trên xe không tính vào khu).
func TestKeepOutSkipsSeatedRider(t *testing.T) {
	r, ids := areaTestRoom(t)
	r.DoWait(func() {
		p := playersByID(r, ids)
		driver, rider := p[1], p[2]
		in := r.zones["garden"].Spawns[0]
		rider.pos, rider.rideOf = in, driver
		if r.keepOutOfFullArea(rider) != "" || rider.pos != in {
			t.Fatal("khách đang ngồi xe không bị dời ra cổng")
		}
		rider.rideOf = nil
	})
}

// Đổi kênh mà chỗ cũ nằm trong khu đã đầy ở kênh mới: đứng ngoài cổng và được báo lý do.
func TestSwitchRoomIntoFullAreaToasts(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "doikenh", "Gà Đổi Kênh", func(*Account) {})
	b, _ := e.seeded(t, "trongvuon2", "Gà Trong Vườn", func(*Account) {})
	old := Zones["garden"].Capacity
	Zones["garden"].Capacity = 1
	t.Cleanup(func() { Zones["garden"].Capacity = old })
	first := e.roomOf(t)
	e.warp(b, "garden") // kênh đầu: vườn đã đủ 1 người
	a.send(msg{"t": "switch_room", "room": "new"})
	a.expect("welcome", 2*time.Second)
	e.warp(a, "garden") // kênh mới: a đứng trong vườn
	a.send(msg{"t": "switch_room", "room": first.ID})
	w := a.expect("welcome", 2*time.Second)
	if w["zone"] == "garden" {
		t.Fatal("đổi về kênh có vườn đầy phải đứng ngoài cổng")
	}
	a.toastHas("đang đông nên bạn đứng ngoài cổng")
}
