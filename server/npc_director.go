package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Đạo diễn làng sống (PLAN_LOGIC_GAME 5–9): chạy 2 lần/giây mỗi phòng, quyết định lời chào, lời mời xe ôm,
// cameo và phản ứng, luôn trong hạn mức (5.2) để NPC vui mà không làm phiền.

const GreetRadius = 6.0

// ---------- Vào game ----------

// onJoinNPC: điểm danh hằng ngày (Cụ Cục trao quà) và lời gọi gà mới của Cụ Cục.
func (r *Room) onJoinNPC(p *Player, now time.Time) {
	if r.Kind != "public" || p.acc.Char == nil {
		return
	}
	r.grantWeeklyTitles(p, now)
	if r.hub.bal.JoinGiftsOff {
		return
	}
	// Tài khoản có từ trước bản cập nhật: nhận bù quà các cấp đã đạt (PLAN_LOGIC_GAME 12).
	if got := p.acc.ClaimLevelRewards(now); len(got) > 0 {
		total := Reward{Items: map[string]int{}, Food: map[string]int{}}
		for _, rw := range got {
			total.Thoc += rw.Thoc
			for id, n := range rw.Items {
				total.Items[id] += n
			}
			for id, n := range rw.Food {
				total.Food[id] += n
			}
		}
		if err := r.hub.store.Save(p.acc); err == nil {
			r.toast(p, "good", fmt.Sprintf("🎁 Quà các cấp đã đạt: %s", rewardText(total)))
		}
	}
	if day, rw, err := p.acc.DoCheckIn(now); err == nil {
		if err := r.hub.store.Save(p.acc); err == nil {
			r.toast(p, "good", fmt.Sprintf("📅 Điểm danh ngày %d: %s", day, rewardText(rw)))
			p.memo.checkinDay = day
		}
	}
	if !p.acc.Tutorial["met_elder"] && r.Region == RegionHypex {
		p.memo.welcomeAt = now.Add(1500 * time.Millisecond)
	} else if p.memo.checkinDay > 0 {
		p.memo.welcomeAt = now.Add(2500 * time.Millisecond)
	}
}

// forgetPlayer gỡ mọi mời mọc/cameo đang nhắm vào người vừa rời phòng.
func (r *Room) forgetPlayer(p *Player) {
	for _, a := range r.actors {
		if a.Target == p {
			if a.Kind == KindXeOm {
				r.xeomRelease(a, false)
			} else {
				a.Target = nil
			}
		}
		for _, c := range a.Cues {
			if c.To == p {
				c.done = true
			}
		}
	}
}

// ---------- Nhịp đạo diễn ----------

func (r *Room) director(now time.Time) {
	for _, p := range r.players {
		if !p.goneAt.IsZero() || p.acc.Char == nil {
			continue
		}
		// Lời gọi gà mới / chào lại sau điểm danh.
		if !p.memo.welcomeAt.IsZero() && now.After(p.memo.welcomeAt) {
			p.memo.welcomeAt = time.Time{}
			if !p.acc.Tutorial["met_elder"] {
				r.sayFrom("ga_cu_cuc", "elder.welcome_new", kvFor(p), p)
			} else if p.memo.checkinDay > 0 {
				kv := kvFor(p)
				kv["n"] = strconv.Itoa(p.memo.checkinDay)
				r.sayFrom("ga_cu_cuc", "elder.checkin", kv, p)
			}
		}
		r.tutorialTick(p, now)
		if p.carrier != nil {
			continue
		}
		if p.pos.Dist(p.memo.stillPos) > 0.6 {
			p.memo.stillPos, p.memo.stillAt = p.pos, now
		}
		r.greetTick(p, now)
		r.hungerTick(p, now)
		r.nestBuddy(p, now)
		if r.world != nil && p.zone == r.world.Outdoor && p.ride == nil && p.rideOf == nil {
			r.cameoTick(p, now)
		}
	}
	if r.world != nil && r.Region == RegionHypex {
		r.offerTick(now)
		r.showTick(now)
	}
}

// ---------- Hướng dẫn người mới (PLAN_LOGIC_GAME 6) ----------

func (r *Room) tutorialTick(p *Player, now time.Time) {
	if p.acc.TutorialClaimed {
		if p.memo.markSent != "done" {
			p.memo.markSent = "done"
			p.sess.Send(map[string]any{"t": "mark", "step": "", "npc": ""})
		}
		return
	}
	next := p.acc.NextTutorial()
	if next == nil {
		// Xong mọi bước: Cụ Cục gửi thưởng qua Chim Chíp, không bắt quay về sảnh.
		if err := p.acc.ClaimTutorial(now, r.hub.bal); err == nil {
			if err := r.hub.store.Save(p.acc); err == nil {
				r.sayFrom("ga_cu_cuc", "elder.tut_done", kvFor(p), p)
				r.toast(p, "good", fmt.Sprintf("🎁 Hoàn thành hướng dẫn: +%d thóc, 3 cám, 1 trứng thường!", r.hub.bal.TutorialReward))
				p.sess.Send(map[string]any{"t": "me", "me": p.acc.View(now, r.hub.bal)})
				id := p.acc.ID
				go func() {
					if r.hub.store.Social.AddMail(id, "tutorial", "🐔 Cụ Cục: Giỏi lắm! Thưởng hướng dẫn đã vào túi cháu rồi. Giờ thì tự do khám phá làng nhé!") {
						r.hub.mailPing(id)
					}
				}()
			}
		}
		return
	}
	if p.memo.markSent != next.ID {
		p.memo.markSent = next.ID
		p.sess.Send(map[string]any{"t": "mark", "step": next.ID, "text": next.Text, "npc": next.NPC, "area": next.Area})
		// NPC của bước kế nói câu hướng dẫn khi người chơi ở gần.
		switch next.ID {
		case "first_run":
			r.sayFrom("ga_don_khach", "guide.run", kvFor(p), p)
		case "reach_garden":
			if r.roads() != nil {
				p.sess.Send(map[string]any{"t": "say", "id": "chim_chip_bus", "name": "Chim Chíp", "text": NPCLines["bird.tut_bus"][0], "to": p.PID, "remote": true, "ttl": 7})
			}
		}
	}
}

// ---------- Lời chào của NPC cố định (PLAN_LOGIC_GAME 9.4) ----------

// greetLine chọn câu chào theo NPC và trạng thái người chơi. strong=true: câu có việc (bước hướng dẫn,
// giun hiếm, phiếu giảm giá, trứng…) được nói lại sau 10 phút; câu xã giao chỉ một lần mỗi ngày.
func (r *Room) greetLine(p *Player, it *Item, now time.Time) (key string, kv map[string]string, strong bool) {
	a := p.acc
	kv = kvFor(p)
	next := a.NextTutorial()
	step := ""
	if next != nil && !a.TutorialClaimed {
		step = next.ID
	}
	switch it.Kind {
	case "npc_elder":
		if a.HungerStage(r.hub.bal) >= 1 {
			return "elder.hungry", kv, true
		}
		return "elder.welcome_back", kv, false
	case "npc_guide":
		if step == "first_run" {
			return "guide.run", kv, true
		}
	case "npc_pig":
		if step == "first_dig" || step == "reach_garden" {
			return "pig.tut_dig", kv, true
		}
		return "pig.garden", kv, false
	case "shop_sell":
		for _, w := range AllWorms {
			if a.Worms[w.ID] > 0 && wormTier(&w) >= 2 {
				kv["giun"] = w.Name
				return "pig.rare", kv, true
			}
		}
		if step == "first_sell" {
			return "pig.tut_sell", kv, true
		}
		kv["giun"] = wormByID(HotWorm(r.Region, now)).Name
		return "pig.greet", kv, false
	case "shop_food":
		if step == "first_meal" {
			return "food.tut_eat", kv, true
		}
		return "food.greet", kv, false
	case "shop_fashion":
		if a.Equipped["head"] == "vuong_mien" {
			return "fashion.crown", kv, false
		}
		return "fashion.greet", kv, false
	case "npc_goat":
		if len(a.OwnedVehicles) == 0 {
			if _, had := a.Coupons[CouponFirstBike]; !had && a.GiveCoupon(CouponFirstBike, 24*time.Hour, now) {
				p.dirty = true
				r.hub.store.Save(a)
				p.sess.Send(map[string]any{"t": "me", "me": a.View(now, r.hub.bal)})
				return "goat.coupon", kv, true
			}
			if a.Thoc >= vehicleByID("xe_dap").Cost {
				return "goat.can_bike", kv, false
			}
		}
		return "goat.greet", kv, false
	case "npc_duck":
		if a.eggCount() > 0 && a.Incubation == nil {
			return "duck.eggs", kv, true
		}
		return "duck.greet", kv, false
	case "npc_cow":
		if a.Level >= 3 {
			return "cow.land", kv, false
		}
		return "cow.greet", kv, false
	case "npc_investor":
		if a.Thoc < LandTiers[1].Cost {
			return "investor.broke", kv, false
		}
		return "investor.greet", kv, false
	case "pk":
		return "dog.greet", kv, false
	case "npc_bird":
		kv["n"] = strconv.Itoa(a.Tokens(now, r.hub.bal))
		kv["eta"] = strconv.Itoa(int(r.nextIntercity(now)))
		return "bird.stop", kv, false
	case "npc_frog":
		return "frog.greet", kv, false
	}
	return "", nil, false
}

func (r *Room) greetTick(p *Player, now time.Time) {
	z := r.zones[p.zone]
	if z == nil || p.acc.Prefs.Chatter == "low" && p.acc.TutorialClaimed {
		return
	}
	day := DayKey(now)
	for i := range z.Items {
		it := &z.Items[i]
		if p.pos.Dist(it.Pos()) > GreetRadius {
			continue
		}
		key, kv, strong := r.greetLine(p, it, now)
		if key == "" {
			continue
		}
		last := p.memo.greetAt[it.ID]
		full := p.acc.NPCSeen[it.ID] != day
		if !full && (!strong || now.Sub(last) < 10*time.Minute) {
			continue
		}
		if !full && p.acc.Prefs.Chatter == "normal" && !strings.Contains(key, "tut") {
			continue
		}
		p.memo.greetAt[it.ID] = now
		if full {
			p.acc.NPCSeen[it.ID] = day
			p.dirty = true
		}
		r.sayFrom(it.ID, key, kv, p)
		return // mỗi nhịp tối đa một lời chào
	}
}

// HotWorm: mỗi ngày Lợn Ủn thu mua giá cao +10% một loài (Thường…Hiếm) của vùng, ai cũng thấy như nhau.
func HotWorm(region string, now time.Time) string {
	set := WormSets[region]
	if len(set) == 0 {
		set = Worms
	}
	h := 0
	for _, c := range DayKey(now) {
		h = h*31 + int(c)
	}
	n := min(3, len(set))
	return set[(h%n+n)%n].ID
}

// ---------- Nghỉ ổ cạnh bạn bè (PLAN_LOGIC_GAME 16) ----------

const NestBuddyRadius = 3.2

// nestBuddy: ngồi ổ cạnh một người bạn đủ một phút thì bạn đồng hành đang đi theo +1 thân thiết, cứ thế mỗi phút.
func (r *Room) nestBuddy(p *Player, now time.Time) {
	buddy := false
	if p.seat != "" {
		for _, q := range r.players {
			if q != p && q.seat != "" && q.pos.Dist(p.pos) <= NestBuddyRadius && r.hub.store.Social.AreFriends(p.acc.ID, q.acc.ID) {
				buddy = true
				break
			}
		}
	}
	switch {
	case !buddy:
		p.memo.buddySince = time.Time{}
	case p.memo.buddySince.IsZero():
		p.memo.buddySince = now
	case now.Sub(p.memo.buddySince) >= time.Minute:
		p.memo.buddySince = now
		if c := p.acc.companion(p.acc.Following); c != nil {
			c.Affinity++
			p.dirty = true
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "buddy"})
		}
	}
}

// ---------- Đói: NPC gần nhất nhắc ----------

func (r *Room) hungerTick(p *Player, now time.Time) {
	if p.acc.HungerStage(r.hub.bal) < 1 || now.Sub(p.memo.hungryAt) < 3*time.Minute {
		return
	}
	if it := r.nearestNPC(p, 15); it != nil {
		p.memo.hungryAt = now
		r.sayFrom(it.ID, "react.hungry", kvFor(p), p)
	}
}

var npcKinds = map[string]bool{"npc_elder": true, "npc_pig": true, "shop_sell": true, "shop_food": true, "shop_fashion": true, "npc_goat": true,
	"npc_duck": true, "npc_cow": true, "npc_bird": true, "npc_frog": true, "pk": true, "npc_guide": true, "npc_investor": true}

func (r *Room) nearestNPC(p *Player, within float64) *Item {
	var best *Item
	bd := within
	for _, z := range r.zones {
		for i := range z.Items {
			it := &z.Items[i]
			if !npcKinds[it.Kind] {
				continue
			}
			if d := p.pos.Dist(it.Pos()); d < bd {
				best, bd = it, d
			}
		}
	}
	return best
}

// npcEvent cho NPC quanh người chơi phản ứng với một sự kiện (đào được giun hiếm, giun thoát, lên cấp…).
func (r *Room) npcEvent(p *Player, kind string, w *Worm) {
	now := time.Now()
	if p.acc.Prefs.Chatter == "low" && kind != "levelup" {
		return
	}
	switch kind {
	case "dug":
		if w == nil || wormTier(w) < 2 || now.Sub(p.memo.reactAt) < 20*time.Second {
			return
		}
		p.memo.reactAt = now
		kv := kvFor(p)
		kv["giun"] = w.Name
		if it := r.nearestNPC(p, 15); it != nil {
			r.sayFrom(it.ID, "react.rare", kv, p)
		}
		if r.world != nil && r.Kind == "public" && now.Sub(p.memo.cameoAt) > 2*time.Minute && r.cameoCount() < 3 && r.rnd() < 0.6 {
			r.reporter(p, w, now)
		}
	case "escaped":
		if now.Sub(p.memo.reactAt) < 20*time.Second {
			return
		}
		p.memo.reactAt = now
		if it := r.nearestNPC(p, 12); it != nil {
			r.sayFrom(it.ID, "react.escape", kvFor(p), p)
		}
	case "levelup":
		kv := kvFor(p)
		kv["cap"] = strconv.Itoa(p.acc.Level)
		r.sayFrom("ga_cu_cuc", "elder.levelup", kv, p)
	case "lostpk":
		p.memo.lostPKAt = now
		if it := r.nearestNPC(p, 20); it != nil && !p.acc.Prefs.NoTease {
			r.sayFrom(it.ID, "react.lostpk", kvFor(p), p)
		}
	}
}

// ---------- Cameo (PLAN_LOGIC_GAME 9.2) ----------

func (r *Room) cameoCount() int {
	n := 0
	for _, a := range r.actors {
		if a.Kind == KindCameo && a.Role != "ga_con" {
			n++
		}
	}
	return n
}

func (r *Room) cameoTick(p *Player, now time.Time) {
	if r.cameoCount() >= 2 || p.dig != nil || p.match != nil || p.trade != nil {
		return
	}
	// Hụt hơi 3 lần trong 2 phút: cụ dưỡng sinh khuyên chạy chậm lại.
	var recent []time.Time
	for _, t := range p.memo.lockouts {
		if now.Sub(t) < 2*time.Minute {
			recent = append(recent, t)
		}
	}
	p.memo.lockouts = recent
	if len(recent) >= 3 && r.cameoOK(p, now) {
		r.elderExercise(p, now)
		return
	}
	hub := r.world.Hub
	stage := p.acc.HungerStage(r.hub.bal)
	switch {
	case r.teaseOK(p, now) && p.acc.Level >= 2 && p.moving && r.rnd() < 0.02:
		r.richKid(p, now)
	case r.teaseOK(p, now) && (stage >= 1 || now.Sub(p.memo.lostPKAt) < time.Minute || now.Sub(p.memo.stillAt) > time.Minute) && r.rnd() < 0.05:
		r.momAndChick(p, now)
	case r.cameoOK(p, now) && p.pos.Dist(hub) < 30 && r.rnd() < 0.02:
		r.shipperAsks(p, now)
	case r.cameoOK(p, now) && p.acc.Mana < 30 && r.zones["market"] != nil && p.pos.Dist(*r.zones["market"].Entry) > 60 && r.rnd() < 0.04:
		r.vendor(p, now)
	}
}

// ---------- Mời xe ôm ----------

func (r *Room) offerTick(now time.Time) {
	targeted := map[*Player]bool{}
	for _, a := range r.actors {
		if a.Target != nil {
			targeted[a.Target] = true
		}
	}
	walkers := r.outdoorWalkers()
	for _, a := range r.actors {
		if a.Kind != KindXeOm || (a.State != "cruise" && a.State != "stand") {
			continue
		}
		reach := 25.0
		if a.Stand {
			reach = 7
		}
		var best *Player
		bd := reach
		for _, p := range walkers {
			if targeted[p] || !r.offerAllowed(p, now) {
				continue
			}
			if d := p.pos.Dist(a.Pos); d < bd {
				best, bd = p, d
			}
		}
		if best != nil {
			a.State, a.Target = "approach", best
			targeted[best] = true
		}
	}
}

// ---------- Trận biểu diễn ở Sân tỷ thí (PLAN_LOGIC_GAME 9.3) ----------

type showCue struct {
	T    float64 `json:"t"`
	Who  string  `json:"who"` // ref | f0 | f1 | c0..c9 (khán giả)
	Text string  `json:"text"`
}

type arenaShow struct {
	ID     string    `json:"id"`
	Where  string    `json:"where"` // arena · park (sân khấu ngày hội)
	Start  time.Time `json:"-"`
	Dur    float64   `json:"dur"`
	Seed   uint32    `json:"seed"`
	Names  [2]string `json:"names"`
	Winner int       `json:"winner"`
	Cues   []showCue `json:"cues"`
}

func (s *arenaShow) msg(now time.Time) map[string]any {
	return map[string]any{"t": "scene", "kind": "arena_show", "id": s.ID, "where": s.Where, "seed": s.Seed, "names": s.Names, "winner": s.Winner,
		"cues": s.Cues, "dur": s.Dur, "since": now.Sub(s.Start).Seconds()}
}

var FighterNames = [][2]string{{"Gà Tía Mặt Sắt", "Gà Ô Lì Đòn"}, {"Gà Chọi Mỏ Thép", "Gà Ri Lanh Chanh"}, {"Gà Lông Vàng", "Gà Mào Lửa"}}

func (r *Room) arenaBusy() bool {
	for _, p := range r.players {
		if p.match != nil && p.zone == "arena" {
			return true
		}
	}
	return false
}

// showWhere: trận biểu diễn ở Sân tỷ thí; ngày hội cộng đồng mở thì dời sang sân khấu ở Ao và vườn (9.3).
func (r *Room) showWhere() string {
	if r.zones["park"] != nil && r.hub.festival.Snapshot().Unlocked {
		return "park"
	}
	return "arena"
}

func (r *Room) showTick(now time.Time) {
	where := r.showWhere()
	arena := r.zones[where]
	if arena == nil {
		return
	}
	if s := r.show; s != nil {
		if s.Where == "arena" && r.arenaBusy() || now.Sub(s.Start).Seconds() > s.Dur {
			r.broadcastRoom(nil, map[string]any{"t": "scene_end", "id": s.ID})
			r.show = nil
			r.showNext = now.Add(4 * time.Minute)
		}
		return
	}
	if now.Before(r.showNext) || where == "arena" && r.arenaBusy() {
		return
	}
	center := Vec{arena.CX, arena.CZ}
	near := false
	for _, p := range r.players {
		if p.pos.Dist(center) < 60 {
			near = true
		}
	}
	if !near {
		return
	}
	r.show = r.buildShow(now)
	r.show.Where = where
	r.broadcastRoom(nil, r.show.msg(now))
}

func (r *Room) buildShow(now time.Time) *arenaShow {
	names := FighterNames[int(r.rnd()*float64(len(FighterNames)))%len(FighterNames)]
	s := &arenaShow{ID: fmt.Sprintf("show%d", now.UnixNano()), Start: now, Dur: 66, Seed: r.rng.Uint32(), Names: names, Winner: int(r.rnd() * 2)}
	pick := func(key string, f string) string {
		lines := NPCLines[key]
		return fillLine(lines[int(r.rnd()*float64(len(lines)))%len(lines)], map[string]string{"f": f})
	}
	s.Cues = append(s.Cues, showCue{1, "ref", pick("ref.start", "")})
	for t := 6.0; t < 50; t += 3 + r.rnd()*3 {
		fav := names[int(r.rnd()*2)]
		key := []string{"crowd.cheer", "crowd.bet", "crowd.mock"}[int(r.rnd()*3)%3]
		s.Cues = append(s.Cues, showCue{math.Round(t*10) / 10, "c" + strconv.Itoa(int(r.rnd()*10)), pick(key, fav)})
	}
	s.Cues = append(s.Cues, showCue{55, "ref", pick("ref.end", names[s.Winner])})
	s.Cues = append(s.Cues, showCue{58, "c" + strconv.Itoa(int(r.rnd()*10)), pick("crowd.end", "")})
	s.Cues = append(s.Cues, showCue{61, "ref", pick("ref.invite", "")})
	return s
}

// nextIntercity: giây tới chuyến xe liên trại kế tiếp (lịch cố định theo BusEvery).
func (r *Room) nextIntercity(now time.Time) float64 {
	every := r.hub.bal.BusEvery.Seconds()
	if every <= 0 {
		return 0
	}
	t := float64(now.UnixMilli()) / 1000
	return every - math.Mod(t, every)
}

// ---------- Hành động: xe buýt, xe ôm, trả lời NPC ----------

func (r *Room) npcAct(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn txnFunc) {
	now := time.Now()
	b := r.hub.bal
	rn := r.roads()
	switch m.A {
	case "bus_stops":
		if rn == nil || len(rn.Stops) == 0 {
			fail(errors.New("Vùng này chưa có xe buýt vòng làng."))
			return
		}
		here := r.stopAt(p)
		var stops []map[string]any
		for i, st := range rn.Stops {
			eta, _ := r.busETA(i, now)
			stops = append(stops, map[string]any{"id": st.ID, "title": st.Title, "eta": math.Ceil(eta), "x": st.Pos.X, "z": st.Pos.Z})
		}
		reply(ActResult{OK: true, Data: map[string]any{"stops": stops, "here": here, "tokens": p.acc.Tokens(now, b),
			"nextToken": math.Round(p.acc.NextTokenIn(now, b)), "fare": fareAfter(p.acc, VillageFare), "riding": p.carrier != nil && p.carrier.Kind == KindBus}})

	case "bus_board":
		if rn == nil {
			fail(errors.New("Vùng này chưa có xe buýt vòng làng."))
			return
		}
		idx := r.stopAt(p)
		if idx < 0 {
			fail(errors.New("Đứng ở bến xe buýt (biển 🚌 cạnh cổng mỗi khu) để lên xe."))
			return
		}
		if err := r.busy(p); err != nil {
			fail(err)
			return
		}
		eta, bus := r.busETA(idx, now)
		if eta > 0 || bus == nil {
			fail(fmt.Errorf("Xe tới sau %.0f giây, chờ chút nhé.", math.Ceil(eta)))
			return
		}
		seat := seatOf(bus)
		if seat < 0 {
			fail(errors.New("Xe đầy rồi, chờ chuyến sau nhé."))
			return
		}
		dest := m.To
		if dest != "" {
			ok := false
			for _, st := range rn.Stops {
				ok = ok || st.ID == dest
			}
			if !ok || dest == rn.Stops[idx].ID {
				fail(errors.New("Chọn bến xuống khác bến đang đứng."))
				return
			}
		}
		txn(func() (ActResult, error) {
			token, paid, err := p.acc.payRide(VillageFare, "village_bus", bus.ID, now, b)
			if err != nil {
				return ActResult{}, fmt.Errorf("Hết lượt miễn phí (lượt mới sau %s) và không đủ %d thóc.", humanDur(time.Duration(p.acc.NextTokenIn(now, b))*time.Second), fareAfter(p.acc, VillageFare))
			}
			r.board(p, bus, seat)
			p.busDest = dest
			how := "dùng 1 lượt miễn phí"
			if !token {
				how = fmt.Sprintf("trả %d thóc", paid)
			}
			where := "đi vòng làng"
			if dest != "" {
				where = "tới bến " + zoneTitle(dest)
			}
			return ActResult{Msg: fmt.Sprintf("Đã lên xe %s (%s).", where, how), Data: map[string]any{"bus": bus.ID, "token": token, "paid": paid}}, nil
		})

	case "bus_alight":
		if p.carrier == nil || p.carrier.Kind != KindBus {
			fail(errors.New("Bạn không ngồi xe buýt."))
			return
		}
		p.alightNext = true
		reply(ActResult{OK: true, Msg: "Sẽ xuống ở bến kế tiếp."})

	case "xeom_accept":
		a := r.actors[m.ID]
		switch {
		case a == nil || a.Kind != KindXeOm || a.State != "offer" || a.Target != p:
			fail(errors.New("Lời mời xe ôm đã hết hạn."))
			return
		case p.pos.Dist(a.Pos) > 6:
			fail(errors.New("Lại gần xe ôm hơn."))
			return
		}
		if err := r.busy(p); err != nil {
			fail(err)
			return
		}
		z := r.zones[m.To]
		if z == nil || z.Outdoor || z.Entry == nil {
			fail(errors.New("Chọn một khu trên bản đồ để xe ôm chở tới."))
			return
		}
		route, length, ok := rn.Route(p.pos, m.To)
		if !ok || length < 15 {
			fail(errors.New("Gần lắm rồi, đi bộ vài bước thôi!"))
			return
		}
		fare := xeomFare(p.acc, length, now)
		txn(func() (ActResult, error) {
			if err := p.acc.addThoc(-fare, "xeom_fare", a.ID+":"+m.To, now); err != nil {
				return ActResult{}, fmt.Errorf("Cần %d thóc cho chuyến này.", fare)
			}
			p.sess.Send(map[string]any{"t": "offer_end", "id": a.OfferID})
			r.board(p, a, 0)
			a.State, a.Dest, a.Fare, a.OfferID = "carry", m.To, fare, ""
			a.Path, a.PI = append([]Vec{a.Pos}, route...), 0
			return ActResult{Msg: fmt.Sprintf("Lên xe ôm tới %s: −%d thóc (%.0f m).", z.Title, fare, length), Data: map[string]any{"fare": fare, "meters": math.Round(length)}}, nil
		})

	case "xeom_decline":
		if a := r.actors[m.ID]; a != nil && a.Kind == KindXeOm && a.Target == p && a.State == "offer" {
			r.xeomRelease(a, true)
		}
		reply(ActResult{OK: true})

	case "xeom_call":
		if err := r.busy(p); err != nil {
			fail(err)
			return
		}
		var best *Actor
		bd := 40.0
		for _, a := range r.actors {
			if a.Kind != KindXeOm || a.Target != nil || (a.State != "cruise" && !(a.Stand && a.State == "stand")) {
				continue
			}
			lim := 40.0
			if a.Stand {
				lim = 12
			}
			if d := p.pos.Dist(a.Pos); d < bd && d < lim {
				best, bd = a, d
			}
		}
		if best == nil {
			fail(errors.New("Không có xe ôm nào gần đây. Ra bến xe ôm cây đa ở quảng trường nhé!"))
			return
		}
		best.State, best.Target = "approach", p
		p.memo.offerAt = now
		reply(ActResult{OK: true, Msg: "🛵 " + best.Name + " đang chạy tới đón bạn!"})

	case "xeom_leave":
		if p.carrier == nil || p.carrier.Kind != KindXeOm {
			fail(errors.New("Bạn không ngồi xe ôm."))
			return
		}
		a := p.carrier
		r.unboard(p, "Đã xuống xe ôm.")
		a.State, a.Until, a.Target = "rest", now.Add(10*time.Second), nil
		reply(ActResult{OK: true})

	case "npc_choice":
		r.npcChoice(p, m.ID, m.I, txn, fail)
	}
}
