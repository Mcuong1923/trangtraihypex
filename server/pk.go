package main

import (
	"errors"
	"fmt"
	"log"
	"time"
)

// PK mini-game (mục 10): lời mời có thể từ chối, máy chủ chốt kết quả, không thưởng tài sản theo thắng thua.

type Invite struct {
	ID       string
	Mode     string
	From, To *Player
	Exp      time.Time
}

type Match struct {
	ID       string
	Mode     string // "peck" | "coin"
	A, B     *Player
	Bot      bool // B là Gà Máy
	State    string
	CountEnd time.Time
	PlayEnd  time.Time
	ScoreA   int
	ScoreB   int
	lastTap  [2]time.Time
	BotRate  float64
	// úp ngửa
	Round   int
	NextAt  time.Time
	Waiting bool // đang chờ bên chọn khoá mặt
}

var PKModes = map[string]string{"peck": "Mổ nhanh", "coin": "Úp ngửa"}

func (m *Match) side(p *Player) int {
	if p == m.A {
		return 0
	}
	return 1
}

func (m *Match) chooser() int { return m.Round % 2 } // luân phiên bên chọn trước

func (m *Match) send(r *Room, v map[string]any) {
	m.A.sess.Send(v)
	if !m.Bot {
		m.B.sess.Send(v)
	}
}

func (m *Match) names() (string, string) {
	if m.Bot {
		return m.A.acc.Char.Name, "Gà Máy"
	}
	return m.A.acc.Char.Name, m.B.acc.Char.Name
}

func (r *Room) pkReady(p *Player) error {
	switch {
	case p.zone != "arena":
		return errors.New("PK diễn ra ở Sân tỷ thí.")
	case p.match != nil:
		return errors.New("Bạn đang trong một trận.")
	case p.ride != nil || p.rideOf != nil:
		return errors.New("Không PK khi đang trên xe.")
	case p.acc.HungerStage(r.hub.bal) >= 4:
		return errors.New("Kiệt sức: không nhận PK mới. Hãy ăn trước.")
	}
	return nil
}

func (r *Room) pkAct(p *Player, m *inMsg, reply func(ActResult), fail func(error)) {
	now := time.Now()
	switch m.A {
	case "pk_bot":
		if _, ok := PKModes[m.Mode]; !ok {
			fail(errors.New("Chế độ không hợp lệ."))
			return
		}
		if nearKind(r.zones[p.zone], p.pos, "pk") == nil {
			fail(errors.New("Gặp Chó Mực ở Sân tỷ thí để đấu với Gà Máy."))
			return
		}
		if err := r.pkReady(p); err != nil {
			fail(err)
			return
		}
		r.startMatch(p, nil, m.Mode, now)
		reply(ActResult{OK: true})

	case "pk_invite":
		t := r.byPID(m.Target)
		if _, ok := PKModes[m.Mode]; !ok || t == nil || t == p {
			fail(errors.New("Không tìm thấy người chơi."))
			return
		}
		if err := r.pkReady(p); err != nil {
			fail(err)
			return
		}
		if r.hub.store.Social.Blocked(p.acc.ID, t.acc.ID) {
			fail(ErrBlocked)
			return
		}
		if !r.hub.store.Social.Privacy(t.acc.ID).AllowPK {
			fail(errors.New(t.acc.Char.Name + " đang tắt nhận lời mời PK."))
			return
		}
		if t.zone != p.zone || t.match != nil || t.invite != nil {
			fail(errors.New(t.acc.Char.Name + " đang bận hoặc không ở Sân tỷ thí."))
			return
		}
		if at, ok := p.lastInvited[t.PID]; ok && now.Sub(at) < 10*time.Second {
			fail(errors.New("Chờ chút rồi mời lại người này."))
			return
		}
		r.cancelInvitesFor(p)
		inv := &Invite{ID: fmt.Sprintf("inv%d", now.UnixNano()), Mode: m.Mode, From: p, To: t, Exp: now.Add(PKInviteTTL)}
		r.invites[inv.ID] = inv
		p.invite, t.invite = inv, inv
		p.lastInvited[t.PID] = now
		t.sess.Send(map[string]any{"t": "pk_invite", "id": inv.ID, "from": p.acc.Char.Name, "mode": m.Mode, "modeName": PKModes[m.Mode], "ttl": PKInviteTTL.Seconds()})
		reply(ActResult{OK: true, Msg: "Đã gửi lời mời " + PKModes[m.Mode] + " tới " + t.acc.Char.Name})

	case "pk_accept", "pk_decline":
		inv := r.invites[m.ID]
		if inv == nil || inv.To != p || now.After(inv.Exp) {
			fail(errors.New("Lời mời đã hết hạn."))
			return
		}
		r.dropInvite(inv)
		if m.A == "pk_decline" {
			inv.From.sess.Send(map[string]any{"t": "toast", "kind": "warn", "text": p.acc.Char.Name + " đã từ chối lời mời PK."})
			reply(ActResult{OK: true})
			return
		}
		// Chốt điều kiện đủ sức của cả hai ở đầu trận (mục 7.4).
		for _, x := range []*Player{inv.From, p} {
			if err := r.pkReady(x); err != nil {
				fail(fmt.Errorf("%s: %v", x.acc.Char.Name, err))
				return
			}
		}
		r.startMatch(inv.From, p, inv.Mode, now)
		reply(ActResult{OK: true})

	case "pk_tap":
		mt := p.match
		if mt == nil || mt.ID != m.ID || mt.Mode != "peck" || mt.State != "play" {
			return
		}
		i := mt.side(p)
		// Chỉ nhận đầu vào trong cửa sổ trận và không nhanh hơn PeckMinGap (chống auto-click cơ bản).
		if now.Before(mt.CountEnd) || now.After(mt.PlayEnd.Add(150*time.Millisecond)) || now.Sub(mt.lastTap[i]) < PeckMinGap {
			return
		}
		mt.lastTap[i] = now
		if i == 0 {
			mt.ScoreA++
		} else {
			mt.ScoreB++
		}
		mt.send(r, map[string]any{"t": "pk_score", "id": mt.ID, "a": mt.ScoreA, "b": mt.ScoreB})

	case "pk_pick":
		mt := p.match
		if mt == nil || mt.ID != m.ID || mt.Mode != "coin" || !mt.Waiting || mt.chooser() != mt.side(p) {
			fail(errors.New("Chưa tới lượt bạn chọn."))
			return
		}
		if m.Side != "ngua" && m.Side != "up" {
			fail(errors.New("Chọn Ngửa hoặc Úp."))
			return
		}
		r.flip(mt, m.Side, now)
		reply(ActResult{OK: true})
	}
}

// pkResume dựng lại trạng thái trận cho người vừa nối lại (mục 10.2).
func (r *Room) pkResume(mt *Match, p *Player, now time.Time) map[string]any {
	na, nb := mt.names()
	return map[string]any{"t": "pk_resume", "id": mt.ID, "mode": mt.Mode, "modeName": PKModes[mt.Mode], "you": mt.side(p),
		"names": []string{na, nb}, "state": mt.State, "a": mt.ScoreA, "b": mt.ScoreB, "rounds": CoinRounds,
		"round": mt.Round, "waiting": mt.Waiting, "chooser": mt.chooser(), "duration": PeckDuration.Seconds(),
		"countLeft": max(0, mt.CountEnd.Sub(now).Seconds()), "playLeft": max(0, mt.PlayEnd.Sub(now).Seconds())}
}

func (r *Room) startMatch(a, b *Player, mode string, now time.Time) {
	mt := &Match{ID: fmt.Sprintf("m%d", now.UnixNano()), Mode: mode, A: a, B: b, Bot: b == nil,
		State: "countdown", CountEnd: now.Add(PKCountdown), BotRate: 5 + r.rng.Float64()*2.5}
	if mt.Bot {
		mt.B = &Player{PID: "bot"}
	}
	mt.PlayEnd = mt.CountEnd.Add(PeckDuration)
	r.matches[mt.ID] = mt
	a.match = mt
	if !mt.Bot {
		b.match = mt
	}
	na, nb := mt.names()
	for i, p := range []*Player{a, b} {
		if p == nil {
			continue
		}
		p.sess.Send(map[string]any{"t": "pk_start", "id": mt.ID, "mode": mode, "modeName": PKModes[mode], "you": i,
			"names": []string{na, nb}, "countdown": PKCountdown.Seconds(), "duration": PeckDuration.Seconds(), "rounds": CoinRounds})
	}
	r.broadcastZone(a.zone, nil, map[string]any{"t": "chat", "kind": "system", "text": fmt.Sprintf("⚔️ %s đấu %s với %s!", PKModes[mode], na, nb)})
}

func (r *Room) flip(mt *Match, side string, now time.Time) {
	picks := [2]string{}
	other := map[string]string{"ngua": "up", "up": "ngua"}
	c := mt.chooser()
	picks[c] = side
	picks[1-c] = other[side] // bên kia nhận mặt còn lại
	// Máy chủ tung đồng xu sau khi cả hai đã khoá lựa chọn.
	result := "ngua"
	if r.rng.IntN(2) == 1 {
		result = "up"
	}
	winner := 0
	if picks[1] == result {
		winner = 1
	}
	if winner == 0 {
		mt.ScoreA++
	} else {
		mt.ScoreB++
	}
	mt.Waiting = false
	mt.send(r, map[string]any{"t": "pk_flip", "id": mt.ID, "round": mt.Round, "picks": picks, "result": result, "winner": winner, "a": mt.ScoreA, "b": mt.ScoreB})
	mt.Round++
	mt.NextAt = now.Add(1600 * time.Millisecond)
}

func (r *Room) stepPK(now time.Time) {
	for id, inv := range r.invites {
		if now.After(inv.Exp) {
			r.dropInvite(inv)
			inv.From.sess.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Lời mời PK đã hết hạn."})
			inv.To.sess.Send(map[string]any{"t": "pk_invite_gone", "id": id})
		}
	}
	for _, mt := range r.matches {
		if mt.State == "countdown" && now.After(mt.CountEnd) {
			mt.State = "play"
			mt.send(r, map[string]any{"t": "pk_go", "id": mt.ID})
			if mt.Mode == "coin" {
				mt.NextAt = now
			}
		}
		if mt.State != "play" {
			continue
		}
		switch mt.Mode {
		case "peck":
			if mt.Bot {
				el := now.Sub(mt.CountEnd).Seconds()
				if s := int(mt.BotRate * el); s > mt.ScoreB && now.Before(mt.PlayEnd) {
					mt.ScoreB = s
					mt.send(r, map[string]any{"t": "pk_score", "id": mt.ID, "a": mt.ScoreA, "b": mt.ScoreB})
				}
			}
			if now.After(mt.PlayEnd.Add(200 * time.Millisecond)) {
				r.finishMatch(mt, "")
			}
		case "coin":
			if mt.Waiting || now.Before(mt.NextAt) {
				continue
			}
			if mt.Round >= CoinRounds {
				r.finishMatch(mt, "")
				continue
			}
			mt.Waiting = true
			mt.send(r, map[string]any{"t": "pk_round", "id": mt.ID, "round": mt.Round, "chooser": mt.chooser()})
			if mt.Bot && mt.chooser() == 1 {
				side := "ngua"
				if r.rng.IntN(2) == 1 {
					side = "up"
				}
				mt.NextAt = now.Add(900 * time.Millisecond)
				pick := side
				go func(m *Match) {
					time.Sleep(900 * time.Millisecond)
					r.Do(func() {
						if r.matches[m.ID] == m && m.Waiting {
							r.flip(m, pick, time.Now())
						}
					})
				}(mt)
			}
		}
	}
}

// finishMatch chốt kết quả một lần theo match ID. reason "left" = một bên rời trận.
func (r *Room) finishMatch(mt *Match, leaver string) {
	if r.matches[mt.ID] == nil {
		return
	}
	delete(r.matches, mt.ID)
	winner := -1
	reason := ""
	switch {
	case leaver != "" && mt.State == "countdown":
		reason = "Trận bị huỷ vì một bên rời đi trước khi bắt đầu."
	case leaver != "":
		reason = "Một bên rời trận: bên còn lại thắng."
		if leaver == mt.A.PID {
			winner = 1
		} else {
			winner = 0
		}
	case mt.ScoreA > mt.ScoreB:
		winner = 0
	case mt.ScoreB > mt.ScoreA:
		winner = 1
	}
	cancelled := leaver != "" && mt.State == "countdown"
	mt.send(r, map[string]any{"t": "pk_end", "id": mt.ID, "winner": winner, "a": mt.ScoreA, "b": mt.ScoreB, "reason": reason, "cancelled": cancelled})
	players := []*Player{mt.A}
	if !mt.Bot {
		players = append(players, mt.B)
	}
	for i, p := range players {
		p.match = nil
		if cancelled {
			continue
		}
		p.acc.Stats.PKPlayed++
		p.acc.step("first_social")
		fx := "ko" // người thua ngã KO rồi đứng dậy; không mất đồ hay cấp
		if winner == i {
			p.acc.Stats.PKWins++
			if !mt.Bot {
				r.hub.weeklyPKWin(p.acc, time.Now()) // bảng tuần chỉ tính trận với người thật
			}
			go r.hub.store.Guilds.AddWin(p.acc.ID)
			fx = "win"
		} else if winner == -1 {
			fx = "tie"
		}
		if _, ok := r.players[p.sess]; ok {
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": fx})
			if fx == "ko" {
				r.npcEvent(p, "lostpk", nil)
			}
			if err := r.hub.store.Save(p.acc); err != nil {
				log.Printf("save pk %s: %v", p.acc.ID, err)
			}
		}
	}
}

func (r *Room) endMatchFor(p *Player, reason string) {
	if p.match != nil {
		r.finishMatch(p.match, p.PID)
	}
}

func (r *Room) dropInvite(inv *Invite) {
	delete(r.invites, inv.ID)
	if inv.From.invite == inv {
		inv.From.invite = nil
	}
	if inv.To.invite == inv {
		inv.To.invite = nil
	}
}

func (r *Room) cancelInvitesFor(p *Player) {
	for _, inv := range r.invites {
		if inv.From == p || inv.To == p {
			r.dropInvite(inv)
			other := inv.To
			if other == p {
				other = inv.From
			}
			other.sess.Send(map[string]any{"t": "pk_invite_gone", "id": inv.ID})
		}
	}
}
