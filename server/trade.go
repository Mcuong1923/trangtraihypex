package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"
)

// Giao dịch hai bên (mục 8.2, V1.1): hai người cùng khu, cùng phòng (một goroutine giữ cả hai tài khoản)
// nên hoán đổi là nguyên tử. Mỗi lần sửa đề nghị thì huỷ khoá/xác nhận của cả hai.

type TradeOffer struct {
	Thoc  int            `json:"thoc"`
	Worms map[string]int `json:"worms"`
}

type Trade struct {
	ID        string        `json:"id"`
	A, B      *Player       `json:"-"`
	Offer     [2]TradeOffer `json:"offer"`
	Locked    [2]bool       `json:"locked"`
	Confirmed [2]bool       `json:"confirmed"`
	Open      bool          `json:"open"` // bên B đã nhận lời mời
	Exp       time.Time     `json:"-"`
}

func (t *Trade) side(p *Player) int {
	if p == t.A {
		return 0
	}
	return 1
}

func (t *Trade) view(p *Player) map[string]any {
	return map[string]any{"t": "trade", "id": t.ID, "you": t.side(p), "names": []string{t.A.acc.Char.Name, t.B.acc.Char.Name},
		"offer": t.Offer, "locked": t.Locked, "confirmed": t.Confirmed, "open": t.Open}
}

func (r *Room) sendTrade(t *Trade) {
	t.A.sess.Send(t.view(t.A))
	t.B.sess.Send(t.view(t.B))
}

func (r *Room) closeTrade(t *Trade, why string) {
	if t.A.trade == t {
		t.A.trade = nil
	}
	if t.B.trade == t {
		t.B.trade = nil
	}
	for _, p := range []*Player{t.A, t.B} {
		p.sess.Send(map[string]any{"t": "trade_end", "id": t.ID, "reason": why})
	}
}

// validOffer chỉ cho giun tự đào và thóc, không vượt số đang có.
func validOffer(a *Account, o TradeOffer) error {
	if o.Thoc < 0 || o.Thoc > a.Thoc {
		return fmt.Errorf("Không đủ %d thóc để đưa ra.", o.Thoc)
	}
	for id, n := range o.Worms {
		if wormByID(id) == nil || n < 0 {
			return errors.New("Chỉ giao dịch được giun và thóc.")
		}
		if a.Worms[id] < n {
			return fmt.Errorf("Không đủ %d %s.", n, wormByID(id).Name)
		}
	}
	return nil
}

func (r *Room) tradeAct(p *Player, m *inMsg, reply func(ActResult), fail func(error)) {
	now := time.Now()
	t := p.trade
	switch m.A {
	case "trade_invite":
		o := r.byPID(m.Target)
		switch {
		case o == nil || o == p:
			fail(errors.New("Không tìm thấy người chơi."))
		case p.acc.Level < TradeLevel || o.acc.Level < TradeLevel:
			fail(fmt.Errorf("Cả hai cần từ cấp %d để giao dịch.", TradeLevel))
		case o.zone != p.zone || o.pos.Dist(p.pos) > 4:
			fail(errors.New("Lại gần người đó để giao dịch."))
		case p.trade != nil || o.trade != nil || p.match != nil || o.match != nil:
			fail(errors.New("Một trong hai đang bận."))
		case r.hub.store.Social.Blocked(p.acc.ID, o.acc.ID):
			fail(ErrBlocked)
		default:
			t := &Trade{ID: fmt.Sprintf("tr%d", now.UnixNano()), A: p, B: o, Exp: now.Add(30 * time.Second)}
			for i := range t.Offer {
				t.Offer[i].Worms = map[string]int{}
			}
			p.trade, o.trade = t, t
			o.sess.Send(map[string]any{"t": "trade_invite", "id": t.ID, "from": p.acc.Char.Name})
			reply(ActResult{OK: true, Msg: "Đã mời " + o.acc.Char.Name + " giao dịch."})
		}
	case "trade_accept":
		if t == nil || t.ID != m.ID || t.B != p || t.Open {
			fail(errors.New("Lời mời giao dịch đã hết hạn."))
			return
		}
		t.Open = true
		r.sendTrade(t)
		reply(ActResult{OK: true})
	case "trade_offer":
		if t == nil || t.ID != m.ID || !t.Open {
			fail(errors.New("Không có giao dịch đang mở."))
			return
		}
		o := TradeOffer{Thoc: m.I, Worms: map[string]int{}}
		for k, v := range m.Offer {
			if v > 0 {
				o.Worms[k] = v
			}
		}
		if err := validOffer(p.acc, o); err != nil {
			fail(err)
			return
		}
		t.Offer[t.side(p)] = o
		t.Locked, t.Confirmed = [2]bool{}, [2]bool{} // sửa đề nghị thì cả hai phải khoá lại
		r.sendTrade(t)
		reply(ActResult{OK: true})
	case "trade_lock":
		if t == nil || t.ID != m.ID || !t.Open {
			fail(errors.New("Không có giao dịch đang mở."))
			return
		}
		t.Locked[t.side(p)] = true
		r.sendTrade(t)
		reply(ActResult{OK: true})
	case "trade_confirm":
		if t == nil || t.ID != m.ID || !t.Locked[0] || !t.Locked[1] {
			fail(errors.New("Cả hai phải khoá đề nghị trước khi xác nhận."))
			return
		}
		t.Confirmed[t.side(p)] = true
		if !t.Confirmed[0] || !t.Confirmed[1] {
			r.sendTrade(t)
			reply(ActResult{OK: true, Msg: "Đã xác nhận, chờ bên kia."})
			return
		}
		if err := r.executeTrade(t, now); err != nil {
			r.closeTrade(t, err.Error())
			fail(err)
			return
		}
		r.closeTrade(t, "done")
		reply(ActResult{OK: true, Msg: "Giao dịch thành công!"})
	case "trade_cancel":
		if t != nil && t.ID == m.ID {
			r.closeTrade(t, "Giao dịch đã huỷ.")
		}
		reply(ActResult{OK: true})
	}
}

// executeTrade kiểm lại số dư tại thời điểm chốt rồi hoán đổi; lưu lỗi thì hoàn nguyên cả hai.
func (r *Room) executeTrade(t *Trade, now time.Time) error {
	a, b := t.A.acc, t.B.acc
	if err := validOffer(a, t.Offer[0]); err != nil {
		return errors.New(t.A.acc.Char.Name + ": " + err.Error())
	}
	if err := validOffer(b, t.Offer[1]); err != nil {
		return errors.New(t.B.acc.Char.Name + ": " + err.Error())
	}
	snapA, _ := json.Marshal(a)
	snapB, _ := json.Marshal(b)
	move := func(from, to *Account, o TradeOffer) {
		if o.Thoc > 0 {
			_ = from.addThoc(-o.Thoc, "trade_out", t.ID, now)
			_ = to.addThoc(o.Thoc, "trade_in", t.ID, now)
		}
		for id, n := range o.Worms {
			from.Worms[id] -= n
			to.Worms[id] += n
		}
	}
	move(a, b, t.Offer[0])
	move(b, a, t.Offer[1])
	if err := r.hub.store.Save(a); err != nil {
		restore(a, snapA)
		restore(b, snapB)
		log.Printf("trade save: %v", err)
		return errors.New("Máy chủ tạm không lưu được, giao dịch chưa thực hiện.")
	}
	if err := r.hub.store.Save(b); err != nil {
		restore(a, snapA)
		restore(b, snapB)
		_ = r.hub.store.Save(a)
		log.Printf("trade save: %v", err)
		return errors.New("Máy chủ tạm không lưu được, giao dịch chưa thực hiện.")
	}
	for _, p := range []*Player{t.A, t.B} {
		p.sess.Send(map[string]any{"t": "me", "me": p.acc.View(now, r.hub.bal)})
	}
	return nil
}
