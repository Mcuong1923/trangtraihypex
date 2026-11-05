package main

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Câu cá ở ao (PLAN_LOGIC_GAME 16): đứng ở đầu cầu ao, thả câu, chờ phao chìm rồi giằng co bằng đúng khung
// của đào giun (dig.go). Máy chủ gieo loài cá và seed lúc thả câu, chạy lại mô phỏng để chốt. Cá vào kho,
// bán ở Quầy thức ăn hoặc ăn để hồi độ no.

type FishDef struct {
	Item string  `json:"item"`
	Tier int     `json:"tier"`
	Prob float64 `json:"prob"`
}

var FishTable = []FishDef{
	{"ca_ro", 0, 0.62},
	{"ca_chep", 1, 0.28},
	{"ca_tre", 2, 0.08},
	{"ca_vang", 3, 0.02},
}

const (
	FishBiteMin = 1500 * time.Millisecond // phao chìm sau 1,5–4 giây
	FishBiteMax = 4000 * time.Millisecond
	FishReveal  = 500 * time.Millisecond
	FishXPMult  = 1 // XP như đào giun cùng bậc
)

// fishCooldown: cá lặn lâu hơn ụ đất hồi một chút (bộ live 10 giây); trời mưa cá ăn mồi hơn.
func fishCooldown(b Balance, now time.Time, exhausted bool) time.Duration {
	cd := float64(b.DigCooldown) * 1.25
	if exhausted {
		cd *= 1.5
	}
	if GameClock(now).Rain {
		cd *= RainDigCut
	}
	return time.Duration(cd)
}

// RollFish gieo loài cá. Kiệt sức chỉ câu được cá bậc thấp; đêm cá vàng (bậc cao nhất) ăn mồi gấp rưỡi.
func RollFish(r float64, exhausted, night bool) FishDef {
	if exhausted {
		if r < ExhaustCap {
			return FishTable[0]
		}
		return FishTable[1]
	}
	probs := make([]float64, len(FishTable))
	rest := 0.0
	for i := 1; i < len(FishTable); i++ {
		probs[i] = FishTable[i].Prob
		if night && i == len(FishTable)-1 {
			probs[i] *= NightGlow
		}
		rest += probs[i]
	}
	probs[0] = 1 - rest
	acc := 0.0
	for i, f := range FishTable {
		acc += probs[i]
		if r < acc {
			return f
		}
	}
	return FishTable[len(FishTable)-1]
}

// tugEvents kiểm tra dòng thời gian giữ/thả client gửi lên (đào giun và câu cá dùng chung).
func tugEvents(raw [][2]float64, p TugParams) ([]TugEvent, error) {
	if len(raw) > 400 {
		return nil, errors.New("Dữ liệu giằng co không hợp lệ.")
	}
	events := make([]TugEvent, 0, len(raw))
	last := -1.0
	for _, e := range raw {
		if math.IsNaN(e[0]) || e[0] < last || e[0] > p.MaxSec+1 {
			return nil, errors.New("Dữ liệu giằng co không hợp lệ.")
		}
		last = e[0]
		events = append(events, TugEvent{T: e[0], Hold: e[1] > 0})
	}
	return events, nil
}

// CatchFish cất cá vào kho, trừ sức và tính hồi câu; trả số cấp vừa lên.
func (a *Account) CatchFish(f FishDef, now time.Time, b Balance, cd time.Duration) (int, error) {
	if err := a.addItems(map[string]int{f.Item: 1}); err != nil {
		return 0, err
	}
	a.FishReadyAt = now.Add(cd)
	a.Mana = math.Max(0, a.Mana-b.ManaPerDig)
	a.FishCaught++
	return a.addXP(DigXP[f.Tier] * FishXPMult), nil
}

// MissFish: cá sổng (hoặc bỏ dở) vẫn tính hồi câu và tốn sức, có chút XP an ủi.
func (a *Account) MissFish(now time.Time, b Balance, cd time.Duration) int {
	a.FishReadyAt = now.Add(cd)
	a.Mana = math.Max(0, a.Mana-b.ManaPerDig)
	return a.addXP(EscapeXP)
}

// abandonFish: rời đi, mất kết nối hoặc thả câu lại giữa chừng thì tính như cá sổng (chống gieo lại tìm cá vàng).
func (r *Room) abandonFish(p *Player) {
	f := p.fish
	if f == nil {
		return
	}
	p.fish = nil
	p.acc.FishReadyAt = time.Now().Add(fishCooldown(r.hub.bal, time.Now(), f.Exhausted))
	p.dirty = true
	r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "fish_end"})
}

func (r *Room) fishAct(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn func(func() (ActResult, error))) {
	now, b := time.Now(), r.hub.bal
	zone := r.zones[p.zone]
	switch m.A {
	case "fish_start":
		if p.ride != nil || p.rideOf != nil || p.carrier != nil {
			fail(errors.New("Xuống xe rồi mới câu cá được."))
			return
		}
		it := zone.Item(m.ID)
		if it == nil || it.Kind != "fish" || p.pos.Dist(it.Pos()) > it.R+InteractSlack {
			fail(errors.New("Ra đầu cầu ao (Ao và vườn) để câu cá."))
			return
		}
		r.abandonFish(p)
		r.abandonDig(p)
		switch {
		case now.Before(p.acc.FishReadyAt):
			fail(fmt.Errorf("Cá còn lặn, thả câu lại sau %.0f giây.", math.Ceil(p.acc.FishReadyAt.Sub(now).Seconds())))
			return
		case p.acc.StorageCount() >= StorageCap:
			fail(fmt.Errorf("Kho đầy (%d/%d). Bán hoặc ăn bớt rồi câu tiếp.", p.acc.StorageCount(), StorageCap))
			return
		}
		if p.seat != "" {
			r.stand(p)
		}
		exhausted := p.acc.HungerStage(b) >= 4
		f := RollFish(r.rng.Float64(), exhausted, GameClock(now).Night)
		params := ScaleTug(TugTiers[f.Tier], b.TugSpeed)
		bite := FishBiteMin + time.Duration(r.rng.Float64()*float64(FishBiteMax-FishBiteMin))
		bite = time.Duration(float64(bite) / b.TugSpeed)
		reveal := time.Duration(float64(FishReveal) / b.TugSpeed)
		s := &DigSession{ID: fmt.Sprintf("fish%d", now.UnixNano()), Spot: it.ID, Start: now, TugAt: now.Add(bite + reveal),
			Seed: r.rng.Uint32(), Params: params, Exhausted: exhausted, Fish: &f}
		p.fish = s
		p.in = Vec{}
		r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "cast", "ms": bite.Milliseconds()})
		item := itemByID(f.Item)
		reply(ActResult{OK: true, Data: map[string]any{"id": s.ID, "fish": map[string]any{"id": item.ID, "name": item.Name, "emo": item.Emo},
			"tier": f.Tier, "seed": s.Seed, "params": params, "biteMs": bite.Milliseconds(), "revealMs": reveal.Milliseconds(), "exhausted": exhausted}})

	case "fish_cancel":
		r.abandonFish(p)
		reply(ActResult{OK: true})

	case "fish_finish":
		s := p.fish
		if s == nil || s.ID != m.ID {
			fail(errors.New("Lượt câu đã hết hạn."))
			return
		}
		events, err := tugEvents(m.Ev, s.Params)
		if err != nil {
			fail(err)
			return
		}
		res := SimulateTug(s.Params, TugBursts(s.Seed, s.Params), events)
		if wait := res.End - now.Sub(s.TugAt).Seconds() - 0.6/b.TugSpeed; wait > 0 {
			fail(fmt.Errorf("Cá chưa chịu lên đâu, giữ thêm %.1f giây nữa!", wait))
			return
		}
		p.fish = nil
		cd := fishCooldown(b, now, s.Exhausted)
		item := itemByID(s.Fish.Item)
		txn(func() (ActResult, error) {
			var notes []string
			if res.Caught {
				lv, err := p.acc.CatchFish(*s.Fish, now, b, cd)
				if err != nil {
					return ActResult{}, err
				}
				if lv > 0 {
					r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "levelup", "level": p.acc.Level})
					r.levelUp(p, lv, now, &notes)
				}
				r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "fish_catch", "tier": s.Fish.Tier})
				if s.Fish.Tier >= 2 {
					if it := r.nearestNPC(p, 15); it != nil {
						kv := kvFor(p)
						kv["giun"] = item.Name
						r.sayFrom(it.ID, "react.rare", kv, p)
					}
				}
				return ActResult{Msg: fmt.Sprintf("Câu được %s %s!", item.Emo, item.Name), Data: map[string]any{"caught": true, "item": item.ID,
					"slips": res.Slips, "xp": DigXP[s.Fish.Tier] * FishXPMult, "notes": notes}}, nil
			}
			if lv := p.acc.MissFish(now, b, cd); lv > 0 {
				r.levelUp(p, lv, now, &notes)
			}
			r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "fish_end"})
			return ActResult{Msg: item.Name + " giật đứt mồi rồi lặn mất!", Data: map[string]any{"caught": false, "escaped": true, "item": item.ID,
				"slips": res.Slips, "xp": EscapeXP, "notes": notes}}, nil
		})
	}
}
