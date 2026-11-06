package main

import (
	"fmt"
	"math"
	"time"
)

// Giới hạn người trong khu (sảnh 50, khu khác 25; Zone.Capacity) trên bản đồ liền.
//   - Đi tới cổng khu đầy: dừng ở cổng và vào hàng chờ của khu; có chỗ thì người đứng trước vào trước.
//     Còn đứng gần khu là còn giữ chỗ (không cần giữ phím); đi xa, mất mạng hay lên xe là mất chỗ.
//   - Xuống xe (xe người chơi, xe buýt, xe ôm) hay vào kênh ngay trong khu đầy: được đặt ra ngoài cổng khu.
//     Người bị dời ra không được xếp hàng sẵn: họ định vào mà không qua hàng, nên phải tới cổng và đứng cuối.
// Khu ngoài trời không giới hạn riêng vì phòng đã giới hạn RoomCapacity.

// areaNearDist: đứng cách mép khu không quá khoảng này thì vẫn tính là đang chờ ở cổng.
const areaNearDist = 4.0

// areaWait là một chỗ trong hàng chờ của khu.
type areaWait struct {
	p    *Player
	at   time.Time // lần cuối còn chờ ở cổng; khi đã được báo tới lượt là lúc báo
	told bool      // đã báo "tới lượt bạn": phải vào trong AreaQueueHold kể từ lúc báo
}

// areaAtOrEmpty: khu chứa điểm c trên bản đồ liền, "" với phòng nhà riêng.
func (r *Room) areaAtOrEmpty(c Vec) string {
	if r.world == nil {
		return ""
	}
	return r.world.AreaAt(c)
}

// limitedZone trả về khu có giới hạn người, nil nếu khu không giới hạn.
func (r *Room) limitedZone(area string) *Zone {
	z := r.zones[area]
	if r.world == nil || z == nil || z.Outdoor || z.Capacity <= 0 {
		return nil
	}
	return z
}

// areaOccupants đếm người đang đứng trong khu (không tính khách ngồi trên xe đi ngang và người except).
func (r *Room) areaOccupants(area string, except *Player) int {
	n := 0
	for _, q := range r.players {
		if q != except && q.zone == area && q.rideOf == nil && q.carrier == nil {
			n++
		}
	}
	return n
}

// nearArea: điểm c nằm sát mép khu z (trong khoảng areaNearDist).
func nearArea(z *Zone, c Vec) bool {
	for k := range 8 {
		a := float64(k) * math.Pi / 4
		if z.Contains(Vec{c.X + math.Cos(a)*areaNearDist, c.Z + math.Sin(a)*areaNearDist}) {
			return true
		}
	}
	return false
}

// areaQueue bỏ khỏi hàng những người đã rời phòng, đã vào khu, mất mạng, lên xe, đi xa cổng, hoặc
// được báo tới lượt mà không vào kịp AreaQueueHold.
func (r *Room) areaQueue(area string, now time.Time) []*areaWait {
	z := r.limitedZone(area)
	old := r.areaQ[area]
	q := old[:0]
	for _, w := range old {
		p := w.p
		stay := z != nil && r.players[p.sess] == p && p.zone != area && p.goneAt.IsZero() &&
			p.rideOf == nil && p.carrier == nil && nearArea(z, p.pos) && now.Sub(w.at) <= AreaQueueHold
		if stay {
			q = append(q, w)
		} else if w.told && r.players[p.sess] == p && p.zone != area {
			r.toast(p, "warn", "Bạn đã mất lượt vào khu. Đi tới cổng để xếp hàng lại nhé.")
		}
	}
	clear(old[len(q):]) // không giữ con trỏ tới người đã rời hàng
	if len(q) == 0 {
		delete(r.areaQ, area)
		return nil
	}
	r.areaQ[area] = q
	return q
}

func queueIndex(q []*areaWait, p *Player) int {
	for i, w := range q {
		if w.p == p {
			return i
		}
	}
	return -1
}

// areaFull: bước sang khu đã đủ người thì bị chặn ở cổng (M9). Người tới sau xếp sau người đang chờ:
// chỉ vào được khi số người trong khu cộng số người đứng trước trong hàng còn dưới sức chứa.
// Người giữ vé ở bến của vùng luôn vào được bến, để không bị huỷ chuyến vì bến đông.
func (r *Room) areaFull(p *Player, area string) bool {
	z := r.limitedZone(area)
	if z == nil || area == p.zone || (area == r.stationZone() && p.acc.Booking != nil) {
		return false
	}
	now := time.Now()
	q := r.areaQueue(area, now)
	idx := queueIndex(q, p)
	ahead := idx
	if idx < 0 {
		ahead = len(q)
	}
	if r.areaOccupants(area, p)+ahead < z.Capacity {
		if idx >= 0 {
			r.areaQ[area] = append(q[:idx], q[idx+1:]...)
			clear(q[len(q)-1:])
		}
		return false
	}
	if idx < 0 {
		if r.areaQ == nil {
			r.areaQ = map[string][]*areaWait{}
		}
		r.areaQ[area] = append(q, &areaWait{p: p, at: now})
	} else if !q[idx].told {
		q[idx].at = now
	}
	return true
}

// areaQueuePos: thứ tự (từ 1) của p trong hàng chờ của khu, 0 nếu không chờ.
func (r *Room) areaQueuePos(p *Player, area string) int {
	return queueIndex(r.areaQ[area], p) + 1
}

// stepAreaQueues (mỗi giây): người còn đứng gần cổng thì vẫn giữ chỗ; khu có chỗ thì báo người đầu hàng
// và giữ chỗ cho họ AreaQueueHold.
func (r *Room) stepAreaQueues(now time.Time) {
	for area := range r.areaQ {
		z := r.limitedZone(area)
		q := r.areaQueue(area, now)
		if z == nil || q == nil {
			continue
		}
		free := z.Capacity - r.areaOccupants(area, nil)
		for i, w := range q {
			switch {
			case i < free && !w.told:
				w.told, w.at = true, now
				r.toast(w.p, "good", fmt.Sprintf("Tới lượt bạn vào %s, đi vào trong %d giây nhé.", z.Title, int(AreaQueueHold.Seconds())))
			case !w.told:
				w.at = now // còn đứng gần cổng (đã qua lọc ở areaQueue): vẫn giữ chỗ
			}
		}
	}
}

// keepOutOfFullArea: p vừa xuống xe hoặc vào kênh ngay trong khu đã đủ người (kể cả chỗ giữ cho người
// đang chờ) thì đặt p ra ngoài cổng khu. Trả về tên khu khi phải dời, "" nếu không. Khách còn ngồi xe
// (không tính vào khu) và người đang rời phòng thì để nguyên.
func (r *Room) keepOutOfFullArea(p *Player) string {
	if r.world == nil || p.leaving || p.rideOf != nil || p.carrier != nil {
		return ""
	}
	area := r.world.AreaAt(p.pos)
	z := r.limitedZone(area)
	if z == nil || (area == r.stationZone() && p.acc.Booking != nil) {
		return ""
	}
	q := r.areaQueue(area, time.Now())
	waiting := len(q)
	if i := queueIndex(q, p); i >= 0 {
		waiting = i // p đang chờ: chỉ tính người đứng trước
	}
	if r.areaOccupants(area, p)+waiting < z.Capacity {
		return ""
	}
	p.pos = r.outsideArea(z, p.pos)
	return z.Title
}

// outsideArea: chỗ đứng được ngay ngoài khu, ưu tiên đầu đường vào khu (Zone.Entry), không có thì tìm
// vòng dần ra quanh from.
func (r *Room) outsideArea(z *Zone, from Vec) Vec {
	ok := func(c Vec) bool { return r.world.World.Walkable(c) && r.world.AreaAt(c) == r.world.Outdoor }
	if z.Entry != nil && ok(*z.Entry) {
		return *z.Entry
	}
	for rad := 2.0; rad <= 60; rad += 2 {
		for k := range 16 {
			a := float64(k) * math.Pi / 8
			if c := (Vec{from.X + math.Cos(a)*rad, from.Z + math.Sin(a)*rad}); ok(c) {
				return c
			}
		}
	}
	return from // không tìm được chỗ: để nguyên, không đặt vào tường
}
