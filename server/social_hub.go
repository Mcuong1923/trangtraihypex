package main

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Thao tác xã hội ở cấp phiên (mục 11) và vào/ra nhà riêng (mục 9.1).
// Thứ tự khoá: được lấy social.mu rồi hub.mu, không bao giờ ngược lại.

type homeInvite struct {
	Owner, Guest string
	Exp          time.Time
}

func (h *Hub) sessionOf(id string) *Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[id]
}

// where trả về nơi một tài khoản đang online (để bạn bè xem nếu người đó cho phép).
func (h *Hub) where(id string) (bool, string) {
	s := h.sessionOf(id)
	if s == nil {
		return false, ""
	}
	if r := s.room.Load(); r != nil {
		return true, r.Name
	}
	return true, ""
}

func (h *Hub) mailPing(id string) {
	if o := h.sessionOf(id); o != nil {
		o.Send(map[string]any{"t": "mail_new", "unread": h.store.Social.UnreadMail(id)})
	}
}

func (h *Hub) sendSocial(s *Session) {
	s.Send(map[string]any{"t": "social", "social": h.store.Social.View(s.acc.ID, h.where)})
}

func (h *Hub) ping(id string, v map[string]any) {
	if o := h.sessionOf(id); o != nil {
		o.Send(v)
		h.sendSocial(o)
	}
}

// resolve tìm account ID từ mã người chơi (m.ID) hoặc pid của người cùng phòng (m.Target).
func (h *Hub) resolve(s *Session, m *inMsg) (string, error) {
	if m.Target != "" {
		var id string
		if r := s.room.Load(); r != nil {
			r.DoWait(func() {
				if p := r.byPID(m.Target); p != nil {
					id = p.acc.ID
				}
			})
		}
		if id == "" {
			return "", ErrNotFound
		}
		return id, nil
	}
	if len(m.ID) == 24 { // account ID có sẵn trong danh sách bạn
		return m.ID, nil
	}
	if id, ok := h.store.Social.Lookup(m.ID); ok {
		return id, nil
	}
	return "", ErrNotFound
}

func (h *Hub) social(s *Session, m *inMsg) {
	me := s.acc.ID
	so := h.store.Social
	toast := func(kind, text string) { s.Send(map[string]any{"t": "toast", "kind": kind, "text": text}) }
	fail := func(err error) { toast("warn", err.Error()) }

	switch m.T {
	case "social":
		h.sendSocial(s)

	case "friend_add":
		id, err := h.resolve(s, m)
		if err != nil {
			fail(err)
			return
		}
		friends, err := so.Request(me, id)
		if err != nil {
			fail(err)
			return
		}
		if !friends && so.AddMail(id, "friend", "🐦 "+s.acc.Char.Name+" gửi lời mời kết bạn. Mở bảng Bạn bè để trả lời.") {
			h.mailPing(id)
		}
		if friends {
			toast("good", "Đã là bạn với "+so.Name(id)+"!")
			h.ping(id, map[string]any{"t": "toast", "kind": "good", "text": s.acc.Char.Name + " và bạn đã là bạn bè."})
		} else {
			toast("good", "Đã gửi lời mời kết bạn tới "+so.Name(id)+".")
			h.ping(id, map[string]any{"t": "toast", "kind": "good", "text": "👥 " + s.acc.Char.Name + " muốn kết bạn với bạn."})
		}
		h.sendSocial(s)

	case "friend_respond":
		if err := so.Respond(me, m.ID, m.On); err != nil {
			fail(err)
			return
		}
		if m.On {
			h.ping(m.ID, map[string]any{"t": "toast", "kind": "good", "text": s.acc.Char.Name + " đã nhận lời kết bạn."})
		}
		h.sendSocial(s)

	case "friend_remove":
		if err := so.Unfriend(me, m.ID); err != nil {
			fail(err)
			return
		}
		h.sendSocial(s)
		h.ping(m.ID, map[string]any{"t": "noop"})

	case "block":
		id, err := h.resolve(s, m)
		if err != nil {
			fail(err)
			return
		}
		if err := so.Block(me, id, m.On); err != nil {
			fail(err)
			return
		}
		if m.On {
			toast("good", "Đã chặn "+so.Name(id)+": không nhận tin, lời mời, PK hay chat của người này.")
		}
		h.sendSocial(s)

	case "report":
		id, err := h.resolve(s, m)
		if err != nil {
			fail(err)
			return
		}
		rep := Report{At: time.Now(), Reporter: me, Target: id, Reason: m.Reason, Name: so.Name(id)}
		if r := s.room.Load(); r != nil {
			rep.Room = r.Name
			r.DoWait(func() { rep.Evidence = r.Evidence(id) })
		}
		if err := so.Report(rep); err != nil {
			fail(err)
			return
		}
		toast("good", "Đã gửi báo cáo kèm nhật ký chat của máy chủ. Cảm ơn bạn!")

	case "mail":
		s.Send(map[string]any{"t": "mail", "mail": so.ReadMail(me)})

	case "dm":
		if until, reason, muted := h.store.Mod.Muted(me); muted {
			fail(fmt.Errorf("Bạn đang bị khoá chat đến %s (%s).", until.In(vnZone).Format("15:04 02/01"), reason))
			return
		}
		msg, err := so.SendDM(me, m.ID, m.Text)
		if err != nil {
			fail(err)
			return
		}
		out := map[string]any{"t": "dm", "with": m.ID, "msg": msg}
		s.Send(out)
		if o := h.sessionOf(m.ID); o != nil {
			o.Send(map[string]any{"t": "dm", "with": me, "name": s.acc.Char.Name, "msg": msg})
		}

	case "dm_history":
		s.Send(map[string]any{"t": "dm_history", "with": m.ID, "name": so.Name(m.ID), "msgs": so.History(me, m.ID)})

	case "privacy":
		if err := so.SetPrivacy(me, Privacy{ShowPresence: m.Presence, AllowPK: m.AllowPK, QuietMail: m.On}); err != nil {
			fail(err)
			return
		}
		toast("good", "Đã lưu cài đặt riêng tư.")
		h.sendSocial(s)

	case "invite_room":
		// Mời bạn vào chuồng công cộng mình đang đứng (mục 6.3).
		r := s.room.Load()
		if r == nil || r.Kind != "public" || !so.AreFriends(me, m.ID) {
			fail(errors.New("Chỉ mời được bạn bè vào chuồng công cộng bạn đang ở."))
			return
		}
		o := h.sessionOf(m.ID)
		if o == nil {
			fail(errors.New(so.Name(m.ID) + " đang offline."))
			return
		}
		o.Send(map[string]any{"t": "room_invite", "from": s.acc.Char.Name, "room": r.ID, "roomName": r.Name})
		toast("good", "Đã mời "+so.Name(m.ID)+" vào "+r.Name+".")

	case "home_invite":
		if !so.AreFriends(me, m.ID) {
			fail(errors.New("Chỉ mời được bạn bè về chuồng."))
			return
		}
		o := h.sessionOf(m.ID)
		if o == nil {
			fail(errors.New(so.Name(m.ID) + " đang offline."))
			return
		}
		tok := randHex(12)
		h.mu.Lock()
		h.homeInvites[tok] = homeInvite{Owner: me, Guest: m.ID, Exp: time.Now().Add(HomeInviteTTL)}
		h.mu.Unlock()
		o.Send(map[string]any{"t": "home_invite", "from": s.acc.Char.Name, "owner": me, "token": tok, "ttl": HomeInviteTTL.Seconds()})
		toast("good", "Đã mời "+so.Name(m.ID)+" về chuồng của bạn.")

	case "visit":
		owner := m.Owner
		if owner == "" || owner == "me" {
			owner = me
		}
		if err := h.Visit(s, owner, m.Token); err != nil {
			fail(err)
		}

	case "leave_home":
		h.LeaveHome(s)

	case "contracts":
		s.Send(map[string]any{"t": "contracts", "contracts": h.store.Contracts.List(me), "me": me})

	case "homes":
		type entry struct {
			ID, Name, Access, Why string
			Tier                  int
		}
		var list []entry
		seen := map[string]bool{me: true}
		for _, f := range so.FriendIDs(me) {
			acc, known := h.store.Homes.Access(f)
			if !known {
				acc = "friends"
			}
			if acc != "private" {
				list = append(list, entry{f, so.Name(f), acc, "Bạn bè", 0})
				seen[f] = true
			}
		}
		for id, e := range h.store.Homes.Public() {
			why := "Mở cửa"
			if time.Now().Before(e.PartyEnd) {
				why = "🎉 Đang mở tiệc"
			}
			if !seen[id] && !so.Blocked(me, id) {
				list = append(list, entry{id, e.Name, e.Access, why, e.Tier})
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
		out := []map[string]any{}
		for _, e := range list {
			out = append(out, map[string]any{"id": e.ID, "name": e.Name, "access": e.Access, "why": e.Why})
		}
		s.Send(map[string]any{"t": "homes", "homes": out})
	}
}

// Visit đưa phiên vào nhà riêng của owner nếu có quyền: chủ nhà, khách có lời mời còn hạn,
// bạn bè khi nhà để "bạn bè", mọi người khi "công khai". Người đã chặn nhau luôn bị từ chối.
func (h *Hub) Visit(s *Session, owner, token string) error {
	me := s.acc.ID
	if owner != me {
		if h.store.Social.Blocked(me, owner) {
			return ErrBlocked
		}
		allowed := false
		h.mu.Lock()
		if inv, ok := h.homeInvites[token]; ok && inv.Owner == owner && inv.Guest == me && time.Now().Before(inv.Exp) {
			allowed = true
		}
		h.mu.Unlock()
		if !allowed && h.store.Contracts.WorkerOf(owner, me) {
			allowed = true // thợ đang làm hợp đồng được vào thi công (mục 9.2)
		}
		if !allowed {
			// Nhà chưa từng mở dùng quyền mặc định "bạn bè" như lúc tạo tổ (mục 9.1).
			access, _ := h.store.Homes.Access(owner)
			allowed = access == "public" || (access == "friends" && h.store.Social.AreFriends(me, owner))
		}
		if !allowed {
			return errors.New("Chuồng này đang đóng cửa với bạn.")
		}
	}
	name := s.acc.Char.Name
	if owner != me {
		name = h.store.Social.Name(owner)
	}
	home, err := h.store.Homes.Get(owner, name)
	if err != nil {
		return errors.New("Không mở được chuồng lúc này.")
	}

	cur := s.room.Load()
	if cur != nil && cur.Kind == "home" && cur.Owner == owner {
		return nil
	}
	if cur != nil {
		ok := true
		cur.DoWait(func() { ok = cur.canLeave(s) })
		if !ok {
			return errors.New("Đang trong trận PK, chưa đi được.")
		}
	}
	h.mu.Lock()
	target := h.homes[owner]
	if target == nil {
		target = NewHomeRoom(h, home)
		h.homes[owner] = target
		go target.Run()
	}
	if target.Count() >= HomeCapacity {
		h.mu.Unlock()
		return errors.New("Chuồng đang đủ 8 người, thử lại sau nhé.")
	}
	target.reserve()
	h.mu.Unlock()
	if cur != nil {
		cur.DoWait(func() { cur.remove(s) })
	}
	s.room.Store(target)
	target.DoWait(func() { target.add(s) })
	return nil
}

// LeaveHome đưa phiên từ nhà riêng về sảnh chuồng công cộng gần nhất đã ở.
func (h *Hub) LeaveHome(s *Session) {
	cur := s.room.Load()
	if cur == nil || cur.Kind == "public" {
		return
	}
	ok := true
	cur.DoWait(func() { ok = cur.canLeave(s) })
	if !ok {
		return
	}
	cur.DoWait(func() { cur.remove(s) })
	s.room.Store(nil)
	h.mu.Lock()
	prev := h.lastPublic[s]
	h.mu.Unlock()
	if !s.closed.Load() {
		h.place(s, prev)
	}
}

// closeHome dừng phòng nhà khi không còn ai (kể cả người đang giữ chỗ để vào).
func (h *Hub) closeHome(r *Room) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.Count() > 0 || h.homes[r.Owner] != r {
		return
	}
	delete(h.homes, r.Owner)
	r.Do(func() { r.quit = true })
	for tok, inv := range h.homeInvites {
		if time.Now().After(inv.Exp) {
			delete(h.homeInvites, tok)
		}
	}
}
