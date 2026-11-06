package main

import (
	"fmt"
	"time"
)

// Hàng chờ kênh (PLAN.md: sảnh chuồng đã đầy thì chọn chuồng còn chỗ hoặc xếp hàng để chờ bạn).
//   - Chọn chuồng đầy trong bảng Đổi chuồng (khi đang ở chuồng công cộng): vào hàng chờ, được báo thứ tự.
//   - Mỗi giây hub xét hàng: chuồng có chỗ thì giữ chỗ cho người đầu hàng và tự chuyển họ sang.
//   - Người vào game hay đổi chuồng khác không chen được: chỗ trống dành cho người đang chờ trước.
//   - Tới lượt mà đang PK/trên xe: vẫn đứng đầu hàng, lượt đó nhường người kế tiếp, xong việc thì chuyển.
//   - Mất mạng ngắn vẫn giữ chỗ (nối lại thì chuyển sang phiên mới). Mất chỗ khi huỷ chờ, hết giờ giữ phiên,
//     rời vùng, vào nhà, tự đổi sang chuồng khác hoặc chờ quá RoomQueueHold.

// roomQueueRetry: tới lượt mà đang bận thì sau chừng này mới thử chuyển lại.
const roomQueueRetry = 3 * time.Second

// roomWait là một chỗ trong hàng chờ của một chuồng (hub.roomQ, đọc/ghi khi giữ h.mu).
type roomWait struct {
	s        *Session
	at       time.Time // lúc vào hàng
	pos      int       // thứ tự đã báo gần nhất
	retryAt  time.Time // tới lượt mà đang bận: chưa chuyển trước lúc này
	busyTold bool      // đã báo "đang bận, vẫn giữ chỗ"
}

// queueAheadLocked: số người chờ đứng trước s ở chuồng roomID (cả hàng nếu s không chờ ở đó).
func (h *Hub) queueAheadLocked(roomID string, s *Session) int {
	q := h.roomQ[roomID]
	for i, w := range q {
		if w.s == s {
			return i
		}
	}
	return len(q)
}

// leaveQueuesLocked bỏ s khỏi mọi hàng chờ; trả về true nếu s đang chờ.
func (h *Hub) leaveQueuesLocked(s *Session) bool {
	for id, q := range h.roomQ {
		for i, w := range q {
			if w.s == s {
				h.setQueueLocked(id, append(q[:i], q[i+1:]...))
				return true
			}
		}
	}
	return false
}

// rebindQueueLocked: nối lại sau mất mạng ngắn thì chỗ trong hàng chuyển sang phiên mới.
func (h *Hub) rebindQueueLocked(old, s *Session) {
	for _, q := range h.roomQ {
		for _, w := range q {
			if w.s == old {
				w.s = s
			}
		}
	}
}

func (h *Hub) setQueueLocked(id string, q []*roomWait) {
	if len(q) == 0 {
		delete(h.roomQ, id)
		return
	}
	if h.roomQ == nil {
		h.roomQ = map[string][]*roomWait{}
	}
	h.roomQ[id] = q
}

// enqueueLocked đưa s vào cuối hàng của chuồng roomID (giữ nguyên chỗ nếu đã chờ ở đó), trả về thứ tự từ 1.
func (h *Hub) enqueueLocked(s *Session, roomID string, now time.Time) int {
	if ahead := h.queueAheadLocked(roomID, s); ahead < len(h.roomQ[roomID]) {
		return ahead + 1
	}
	h.leaveQueuesLocked(s) // chỉ chờ một chuồng
	q := append(h.roomQ[roomID], &roomWait{s: s, at: now, pos: len(h.roomQ[roomID]) + 1})
	h.setQueueLocked(roomID, q)
	return len(q)
}

// CancelRoomQueue: người chơi bấm "Huỷ chờ".
func (h *Hub) CancelRoomQueue(s *Session) {
	h.mu.Lock()
	ok := h.leaveQueuesLocked(s)
	h.mu.Unlock()
	text := "Đã rời hàng chờ chuồng."
	if !ok {
		text = "Bạn không còn trong hàng chờ chuồng nào."
	}
	s.Send(map[string]any{"t": "toast", "kind": "good", "text": text})
}

// roomQueueLoop xét hàng chờ mỗi giây (chạy suốt đời hub, như contractJanitor).
func (h *Hub) roomQueueLoop() {
	for range time.Tick(time.Second) {
		h.serveRoomQueues(time.Now())
	}
}

// queueState xét một người chờ: keep = còn giữ chỗ, ready = chuyển được ngay nếu có chỗ, why = lý do mất chỗ
// cần báo (rỗng thì không báo).
func queueState(w *roomWait, target *Room, now time.Time) (keep, ready bool, why string) {
	cur := w.s.room.Load()
	switch {
	case target == nil || w.s.kicked.Load():
		return false, false, ""
	case now.Sub(w.at) > RoomQueueHold:
		return false, false, "Bạn đã chờ quá lâu nên đã rời hàng chờ chuồng."
	case w.s.closed.Load() || cur == nil:
		return true, false, "" // mất mạng/đang chuyển phòng: giữ chỗ, hết giờ giữ phiên thì dropSession gỡ
	case cur == target:
		return false, false, "" // đã vào được chuồng (tự đổi hoặc vừa được chuyển)
	case cur.Kind != "public" || cur.Region != target.Region:
		return false, false, "Bạn đã rời vùng hoặc vào nhà nên đã rời hàng chờ chuồng."
	}
	return true, !now.Before(w.retryAt), ""
}

// serveRoomQueues bỏ người chờ không còn hợp lệ, giữ chỗ cho người chờ sẵn sàng đứng trước nhất khi chuồng có
// chỗ rồi chuyển họ sang, và báo thứ tự mới cho người còn chờ.
func (h *Hub) serveRoomQueues(now time.Time) {
	type note struct {
		s    *Session
		text string
	}
	var notes []note
	var grants []func()
	h.mu.Lock()
	for id, q := range h.roomQ {
		var target *Room
		for _, r := range h.rooms {
			if r.ID == id {
				target = r
			}
		}
		keep := make([]*roomWait, 0, len(q))
		free := 0
		if target != nil {
			free = RoomCapacity - target.Count()
		}
		for _, w := range q {
			ok, ready, why := queueState(w, target, now)
			switch {
			case !ok:
				if why != "" {
					notes = append(notes, note{w.s, why})
				}
			case ready && free > 0:
				free--
				target.reserve() // giữ chỗ ngay để người khác không chen vào giữa lúc chuyển
				from := w.s.room.Load()
				grants = append(grants, func() { h.moveQueued(w, from, target) })
			default:
				keep = append(keep, w)
			}
		}
		for i, w := range keep {
			if w.pos != i+1 {
				w.pos = i + 1
				notes = append(notes, note{w.s, fmt.Sprintf("Hàng chờ %s: bạn đứng thứ %d.", target.Name, w.pos)})
			}
		}
		h.setQueueLocked(id, keep)
	}
	h.mu.Unlock()
	for _, n := range notes {
		n.s.Send(map[string]any{"t": "toast", "kind": "warn", "text": n.text})
	}
	for _, g := range grants {
		go g()
	}
}

// moveQueued chuyển người chờ sang chuồng đã được giữ chỗ. Người đó đã tự đổi chỗ (khác from) hay bị thay phiên
// thì trả chỗ; đang PK/trên xe thì trả chỗ nhưng đưa họ lại đầu hàng để thử lần sau.
func (h *Hub) moveQueued(w *roomWait, from, target *Room) {
	s := w.s
	s.moveMu.Lock()
	defer s.moveMu.Unlock()
	cur := s.room.Load()
	if cur == nil || cur != from || cur == target || s.closed.Load() || s.kicked.Load() {
		target.unreserve()
		return
	}
	if h.moveReserved(s, cur, target) {
		s.Send(map[string]any{"t": "toast", "kind": "good", "text": "Tới lượt rồi! Bạn đã sang " + target.Name + "."})
		return
	}
	h.mu.Lock()
	tell := !w.busyTold
	w.busyTold, w.retryAt, w.pos = true, time.Now().Add(roomQueueRetry), 1
	if h.queueAheadLocked(target.ID, s) == len(h.roomQ[target.ID]) { // chưa tự xếp lại: trả về đầu hàng
		h.setQueueLocked(target.ID, append([]*roomWait{w}, h.roomQ[target.ID]...))
	}
	h.mu.Unlock()
	if tell {
		s.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Tới lượt vào " + target.Name +
			" nhưng bạn đang trong trận PK hoặc trên xe. Bạn vẫn đứng đầu hàng, xong việc sẽ được chuyển."})
	}
}
