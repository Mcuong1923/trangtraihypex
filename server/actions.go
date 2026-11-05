package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"
)

// act xử lý một hành động của người chơi. Hành động chạm tài sản đi qua txn():
// kiểm tra request ID → áp dụng → lưu bền vững → mới trả kết quả; lưu lỗi thì hoàn nguyên.

func (r *Room) act(p *Player, m *inMsg) {
	now := time.Now()
	b := r.hub.bal
	reply := func(res ActResult) {
		p.sess.Send(map[string]any{"t": "res", "rid": m.RID, "a": m.A, "ok": res.OK, "err": res.Err, "msg": res.Msg, "data": res.Data, "me": p.acc.View(now, b)})
	}
	fail := func(err error) { reply(ActResult{Err: err.Error()}) }
	zone := r.zones[p.zone]

	// Tra request ID trước mọi kiểm tra vị trí/trạng thái: gửi lại sau khi mất mạng phải nhận đúng kết quả cũ.
	if prev, ok := p.acc.Remembered(m.RID); ok {
		reply(prev)
		return
	}
	if m.throttled {
		fail(errors.New("Thao tác nhanh quá, chậm lại một chút nhé."))
		return
	}

	// txn bọc một thay đổi tài sản nguyên tử và idempotent.
	txn := func(apply func() (ActResult, error)) {
		if prev, ok := p.acc.Remembered(m.RID); ok {
			reply(prev) // retry sau timeout: trả đúng kết quả cũ, không cộng lần hai
			return
		}
		snap, _ := json.Marshal(p.acc)
		// Panic giữa apply: hoàn nguyên tài khoản rồi báo lỗi, để bản dở dang không bị lưu ở lần Save sau.
		defer func() {
			if e := recover(); e != nil {
				restore(p.acc, snap)
				logPanic("txn "+m.A, e)
				fail(errors.New("Lỗi máy chủ, giao dịch chưa thực hiện."))
			}
		}()
		res, err := apply()
		if err != nil {
			restore(p.acc, snap)
			fail(err)
			return
		}
		res.OK = true
		p.acc.Remember(m.RID, res)
		if err := r.hub.store.Save(p.acc); err != nil {
			log.Printf("save %s: %v", p.acc.ID, err)
			restore(p.acc, snap)
			fail(errors.New("Máy chủ tạm không lưu được, giao dịch chưa thực hiện. Thử lại sau."))
			return
		}
		reply(res)
	}

	// Đang ngồi xe buýt / xe ôm: chỉ được xuống xe, trả lời NPC và vài thao tác không đổi vị trí.
	if p.carrier != nil && !carrierOK[m.A] {
		fail(errors.New("Đang ngồi xe, xuống xe trước đã."))
		return
	}

	switch m.A {
	case "sit":
		it := zone.Item(m.ID)
		if it == nil || it.Kind != "nest" || p.pos.Dist(it.Pos()) > it.R+InteractSlack || p.match != nil || p.ride != nil || p.rideOf != nil {
			fail(errors.New("Lại gần ổ trống để ngồi."))
			return
		}
		for _, q := range r.players {
			if q != p && q.zone == p.zone && q.seat == it.ID {
				fail(errors.New("Ổ này đã có gà ngồi."))
				return
			}
		}
		p.seat = it.ID
		p.pos = it.Pos()
		p.ry = math.Pi
		p.in = Vec{}
		r.abandonDig(p)
		r.broadcastZone(p.zone, nil, map[string]any{"t": "nest", "pid": p.PID, "nest": p.seat, "x": p.pos.X, "z": p.pos.Z})
		reply(ActResult{OK: true})
	case "stand":
		r.stand(p)
		reply(ActResult{OK: true})

	case "go": // đi qua cổng sang khu khác trong cùng chuồng
		if p.match != nil {
			fail(errors.New("Đang trong trận PK."))
			return
		}
		if r.Kind != "public" {
			// Rời nhà riêng / Nhà Hội là chuyển phòng, do hub xử lý ngoài goroutine phòng.
			reply(ActResult{OK: true})
			go r.hub.LeaveHome(p.sess)
			return
		}
		// Trên bản đồ liền chỉ còn lối "Về sảnh an toàn"; mọi nơi khác đi bộ tới bằng đường làng.
		// -devwarp (chỉ dùng chụp ảnh đối chiếu giao diện) cho nhảy thẳng tới khu.
		ok := m.To == ZoneLobby && m.ID == "home" || r.hub.devWarp && m.ID == "devwarp"
		if p.rideOf != nil {
			fail(errors.New("Đang ngồi xe của người khác; xuống xe trước."))
			return
		}
		if m.ID == "home" && (p.ride != nil) {
			fail(errors.New("Cất xe ở điểm đỗ trước khi về sảnh."))
			return
		}
		if m.ID == "home" && r.Region != RegionHypex {
			// "Về sảnh chuồng Hypex" miễn phí từ vùng khác (mục 8.5).
			reply(ActResult{OK: true})
			go r.hub.TransferRegion([]*Session{p.sess}, RegionHypex, []*arrival{{Zone: ZoneLobby}})
			return
		}
		if !ok || r.zones[m.To] == nil {
			fail(errors.New("Đi bộ tới nơi theo đường làng. Mở bản đồ (M) để đặt chỉ đường."))
			return
		}
		r.moveZone(p, m.To)
		reply(ActResult{OK: true})

	case "talk":
		it := zone.Item(m.ID)
		if it == nil || p.pos.Dist(it.Pos()) > it.R+InteractSlack {
			fail(errors.New("Lại gần hơn để trò chuyện."))
			return
		}
		if it.Kind == "npc_elder" && !p.acc.Tutorial["met_elder"] {
			p.acc.step("met_elder")
			p.dirty = true
		}
		r.discover(p, "npc:"+it.ID)
		reply(ActResult{OK: true, Data: map[string]any{"npc": it.ID}})

	case "dig_start":
		if p.ride != nil || p.rideOf != nil {
			fail(errors.New("Không đào giun khi đang trên xe."))
			return
		}
		it := zone.Item(m.ID)
		if it == nil || it.Kind != "dig" || p.pos.Dist(it.Pos()) > it.R+InteractSlack {
			fail(errors.New("Lại gần ụ đất để đào."))
			return
		}
		// Lượt mới thay lượt cũ, nhưng lượt cũ tính như giun thoát (hồi đất) nên không gieo lại được.
		r.abandonDig(p)
		if err := p.acc.CanDig(now, b); err != nil {
			fail(err)
			return
		}
		if p.seat != "" {
			r.stand(p)
		}
		// Máy chủ gieo loài giun và seed giằng co ngay lúc mổ đất (PLAN_LOGIC_GAME 4.3).
		exhausted := p.acc.HungerStage(b) >= 4
		w := RollDigWorm(Regions[r.Region].WormSet, r.rng.Float64(), now, exhausted, digLuck(p.acc), glowFor(now))
		params := ScaleTug(TugTiers[wormTier(w)], b.TugSpeed)
		peck := b.DigPeckMs
		if exhausted {
			peck = peck * 14 / 9 // kiệt sức: mổ chậm hơn
		}
		peckDur := time.Duration(float64(peck)/b.TugSpeed) * time.Millisecond
		revealDur := time.Duration(float64(b.DigRevealMs)/b.TugSpeed) * time.Millisecond
		d := &DigSession{ID: fmt.Sprintf("dig%d", now.UnixNano()), Spot: it.ID, Start: now, TugAt: now.Add(peckDur + revealDur),
			Worm: w, Seed: r.rng.Uint32(), Params: params, Exhausted: exhausted}
		p.dig = d
		p.in = Vec{}
		r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "dig", "tier": wormTier(w), "color": w.Color, "peck": peckDur.Milliseconds()})
		reply(ActResult{OK: true, Data: map[string]any{"id": d.ID, "worm": w, "tier": wormTier(w), "seed": d.Seed, "params": params,
			"peckMs": peckDur.Milliseconds(), "revealMs": revealDur.Milliseconds(), "exhausted": exhausted}})

	case "dig_cancel":
		r.abandonDig(p) // bỏ dở = giun thoát: vẫn tính hồi đất, chống gieo lại tìm giun hiếm
		reply(ActResult{OK: true})

	case "dig_finish":
		d := p.dig
		if d == nil || d.ID != m.ID {
			fail(errors.New("Lượt đào đã hết hạn."))
			return
		}
		events, err := tugEvents(m.Ev, d.Params)
		if err != nil {
			fail(err)
			return
		}
		res := SimulateTug(d.Params, TugBursts(d.Seed, d.Params), events)
		// Không được xong sớm hơn đồng hồ máy chủ (dung sai trễ mạng 0,6 giây, co theo nhịp của bộ cân bằng).
		if wait := res.End - now.Sub(d.TugAt).Seconds() - 0.6/b.TugSpeed; wait > 0 {
			fail(fmt.Errorf("Giun chưa chịu ra đâu, giữ thêm %.1f giây nữa!", wait))
			return
		}
		p.dig = nil
		cd := 1.0
		if d.Exhausted {
			cd = 1.5
		}
		if GameClock(now).Rain {
			cd *= RainDigCut // mưa: giun ngoi lên, ụ hồi nhanh hơn
		}
		txn(func() (ActResult, error) {
			if res.Caught {
				if err := p.acc.CanDig(now, b); err != nil {
					return ActResult{}, err
				}
				loot, notes, lv := p.acc.FinishDigWorm(d.Worm, p.zone, d.Spot, res.Slips == 0, r.rng.Float64(), now, b, cd)
				r.hub.weeklyDug(p.acc, d.Worm, loot.SizeCM, now)
				extra := r.digBonus(p, d, now, &notes)
				r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "dug", "rare": d.Worm.Rare, "color": d.Worm.Color})
				if lv > 0 {
					r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "levelup", "level": p.acc.Level})
					r.levelUp(p, lv, now, &notes)
				}
				r.npcEvent(p, "dug", d.Worm)
				return ActResult{Msg: "Bạn kéo được " + d.Worm.Name + "!", Data: map[string]any{"loot": loot, "notes": notes, "caught": true,
					"slips": res.Slips, "xp": DigXP[wormTier(d.Worm)], "bonus": extra}}, nil
			}
			lv := p.acc.EscapeDig(now, b, cd)
			var notes []string
			if lv > 0 {
				r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "levelup", "level": p.acc.Level})
				r.levelUp(p, lv, now, &notes)
			}
			r.broadcastZone(p.zone, p, map[string]any{"t": "fx", "pid": p.PID, "fx": "escape"})
			r.npcEvent(p, "escaped", d.Worm)
			return ActResult{Msg: d.Worm.Name + " chuồn mất rồi! Lượt này không tính vào hạn đào trong ngày.",
				Data: map[string]any{"caught": false, "escaped": true, "worm": d.Worm.ID, "slips": res.Slips, "xp": EscapeXP, "notes": notes}}, nil
		})

	case "eat_raw":
		txn(func() (ActResult, error) {
			gain, notes, err := p.acc.EatWormRaw(m.ID, b, now)
			if err != nil {
				return ActResult{}, err
			}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			return ActResult{Msg: fmt.Sprintf("Ăn sống %s: +%d độ no. Ngon lành cành đào!", wormByID(m.ID).Name, gain), Data: map[string]any{"notes": notes}}, nil
		})

	case "ration":
		if nearKind(zone, p.pos, "npc_elder") == nil {
			fail(errors.New("Gặp Cụ Cục ở sảnh để xin bữa cứu trợ."))
			return
		}
		txn(func() (ActResult, error) {
			if err := p.acc.ElderRation(now); err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: "Cụ Cục dúi cho một khẩu phần cám. Ăn đi rồi ra vườn đào giun nhé!"}, nil
		})

	case "sell":
		if nearKind(zone, p.pos, "shop_sell") == nil {
			fail(errors.New("Bán giun ở Sạp thu mua giun tại Chợ thóc."))
			return
		}
		txn(func() (ActResult, error) {
			// m.I = số con muốn bán (0/1 = một con); "Bán hết" gửi một lệnh với số lượng.
			got, n, notes, err := p.acc.SellWorms(m.ID, m.I, now)
			if err != nil { // trả sớm: ID lạ thì wormByID là nil, deref sẽ panic
				return ActResult{}, err
			}
			name := wormByID(m.ID).Name
			if n > 1 {
				name = fmt.Sprintf("%d %s", n, name)
			}
			return ActResult{Msg: fmt.Sprintf("Bán %s: +%d thóc", name, got), Data: map[string]any{"notes": notes, "sold": n, "got": got}}, nil
		})

	case "eat_worm":
		txn(func() (ActResult, error) {
			lv, notes, err := p.acc.EatWorm(m.ID, now)
			if err != nil {
				return ActResult{}, err
			}
			w := wormByID(m.ID)
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			if lv > 0 {
				r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "levelup", "level": p.acc.Level})
			}
			return ActResult{Msg: fmt.Sprintf("Chế biến & ăn %s: −%d thóc, +%d XP, +%d mana", w.Name, w.Fee, w.XP, w.Mana),
				Data: map[string]any{"notes": notes, "levelUp": lv}}, nil
		})

	case "buy_food":
		if nearKind(zone, p.pos, "shop_food") == nil {
			fail(errors.New("Mua thức ăn ở Quầy thức ăn tại Chợ thóc."))
			return
		}
		txn(func() (ActResult, error) {
			err := p.acc.BuyFood(m.ID, m.Eat, now)
			if err != nil {
				return ActResult{}, err
			}
			f := foodByID(m.ID)
			if m.Eat {
				r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
				return ActResult{Msg: fmt.Sprintf("%s: −%d thóc, +%d mana", f.Name, f.Cost, f.Mana)}, nil
			}
			return ActResult{Msg: "Đã cất " + f.Name + " vào túi."}, nil
		})

	case "eat_food":
		txn(func() (ActResult, error) {
			if err := p.acc.EatFood(m.ID); err != nil {
				return ActResult{}, err
			}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			return ActResult{Msg: fmt.Sprintf("Đã ăn %s: +%d mana", foodByID(m.ID).Name, foodByID(m.ID).Mana)}, nil
		})

	case "buy_cos":
		if nearKind(zone, p.pos, "shop_fashion") == nil {
			fail(errors.New("Mua đồ ở Quầy thời trang tại Chợ thóc."))
			return
		}
		txn(func() (ActResult, error) {
			if err := p.acc.BuyCosmetic(m.ID, now); err != nil {
				return ActResult{}, err
			}
			r.broadcastLook(p)
			return ActResult{Msg: "Đã mua & mặc " + cosmeticByID(m.ID).Name}, nil
		})

	case "equip", "outfit_save", "outfit_load":
		if p.match != nil {
			fail(errors.New("Không đổi đồ trong trận PK."))
			return
		}
		txn(func() (ActResult, error) {
			var err error
			switch m.A {
			case "equip":
				err = p.acc.Equip(m.Slot, m.ID)
			case "outfit_save":
				err = p.acc.SaveOutfit(m.I)
			default:
				err = p.acc.LoadOutfit(m.I)
			}
			if err == nil && m.A != "outfit_save" {
				r.broadcastLook(p)
			}
			return ActResult{}, err
		})

	case "rescue_start":
		if p.zone != ZoneLobby || nearKind(zone, p.pos, "npc_elder") == nil {
			fail(errors.New("Gặp Gà Cụ Cục ở sảnh để nhận việc."))
			return
		}
		txn(func() (ActResult, error) {
			job, err := p.acc.StartRescue(now)
			if err != nil {
				return ActResult{}, err
			}
			return ActResult{Data: map[string]any{"job": job.ID, "sec": b.RescueSec, "elapsed": now.Sub(job.StartAt).Seconds()}}, nil
		})

	case "rescue_finish":
		txn(func() (ActResult, error) {
			if err := p.acc.FinishRescue(m.ID, now, b); err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Dọn xong ổ rơm: +%d thóc", b.RescueReward)}, nil
		})

	case "claim_tutorial":
		if nearKind(zone, p.pos, "npc_elder") == nil {
			fail(errors.New("Gặp Gà Cụ Cục ở sảnh để nhận thưởng."))
			return
		}
		txn(func() (ActResult, error) {
			if err := p.acc.ClaimTutorial(now, b); err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Hoàn thành hướng dẫn: +%d thóc", b.TutorialReward)}, nil
		})

	case "tutorial_skip":
		// "Tôi biết chơi rồi" (PLAN_LOGIC_GAME 6.1): bỏ dấu hướng dẫn, giữ quà khởi nghiệp, không nhận thưởng hướng dẫn.
		txn(func() (ActResult, error) {
			if p.acc.TutorialClaimed {
				return ActResult{}, errors.New("Cháu đã xong phần hướng dẫn rồi.")
			}
			p.acc.TutorialClaimed, p.acc.TutorialSkipped = true, true
			return ActResult{Msg: "Được, cháu tự khám phá nhé! Lạc đường thì bấm M hoặc quay lại hỏi cụ."}, nil
		})

	case "show":
		txn(func() (ActResult, error) {
			l, err := p.acc.Showcase(m.ID, now, b)
			if err != nil {
				return ActResult{}, err
			}
			w := wormByID(l.Worm)
			// Thẻ lấy từ bản ghi máy chủ, không từ dữ liệu client gửi lên.
			card := map[string]any{"loot": l.ID, "worm": w.ID, "name": w.Name, "rarity": w.Rarity, "color": w.Color, "size": l.SizeCM, "at": l.At, "zone": zoneTitle(l.Zone)}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "show", "color": w.Color})
			r.broadcastRoom(nil, map[string]any{"t": "chat", "kind": "show", "pid": p.PID, "name": p.acc.Char.Name, "tag": PlayerTag(p.acc.ID), "card": card})
			return ActResult{Msg: "Đã khoe " + w.Name + " với chuồng!"}, nil
		})

	case "order":
		if nearKind(zone, p.pos, "board") == nil {
			fail(errors.New("Giao đơn ở Bảng tin trong sảnh."))
			return
		}
		txn(func() (ActResult, error) {
			o, notes, err := p.acc.DeliverOrder(m.ID, now)
			if err != nil {
				return ActResult{}, err
			}
			msg := fmt.Sprintf("%s nhận hàng: +%d thóc", o.NPC, o.Reward)
			if len(o.Gift) > 0 {
				msg += ", " + itemsText(o.Gift)
			}
			return ActResult{Msg: msg, Data: map[string]any{"notes": notes}}, nil
		})

	case "festival_give":
		if nearKind(zone, p.pos, "board") == nil {
			fail(errors.New("Góp cho ngày hội ở Bảng tin trong sảnh."))
			return
		}
		n := m.I
		if !containsInt(FestivalAmounts, n) {
			fail(errors.New("Chọn mức góp 10, 50 hoặc 100 thóc."))
			return
		}
		if p.acc.Thoc < n {
			fail(ErrNotEnough)
			return
		}
		st, err := r.hub.festival.Add(n)
		if err != nil {
			fail(err)
			return
		}
		snap, _ := json.Marshal(p.acc)
		_ = p.acc.addThoc(-n, "festival", st.Name, now)
		p.acc.Festival += n
		notes := p.acc.CheckTitles()
		res := ActResult{OK: true, Msg: fmt.Sprintf("Đã góp %d thóc cho %s. Cảm ơn bạn!", n, st.Name), Data: map[string]any{"notes": notes}}
		p.acc.Remember(m.RID, res)
		if err := r.hub.store.Save(p.acc); err != nil {
			restore(p.acc, snap)
			r.hub.festival.Undo(n)
			fail(errors.New("Máy chủ tạm không lưu được, chưa trừ thóc."))
			return
		}
		go r.hub.BroadcastAll(map[string]any{"t": "festival", "festival": st})
		if st.Unlocked && st.Total-n < st.Goal {
			go r.hub.Announce("🏮 Cả cộng đồng đã góp đủ cho " + st.Name + "! Sảnh mọi chuồng đã treo đèn lồng.")
		}
		reply(res)

	case "title":
		txn(func() (ActResult, error) {
			if err := p.acc.SetTitle(m.ID); err != nil {
				return ActResult{}, err
			}
			r.broadcastLook(p)
			return ActResult{}, nil
		})

	case "home_buy", "home_place", "home_move", "home_pickup", "home_upgrade", "home_build", "home_upgrade_house", "home_access", "home_kick", "home_party", "party_cheer":
		r.homeAct(p, m, reply, fail)

	case "eggs_claim", "incubate", "hatch", "follow", "rename_pet", "care", "book", "cancel_booking",
		"vehicle_buy", "vehicle_out", "vehicle_in", "seat_invite", "seat_accept", "seat_leave", "seat_kick", "drive_region":
		r.v11Act(p, m, reply, fail, txn)

	case "guild_create", "guild_give":
		r.guildAct(p, m, reply, fail, txn)

	case "contract_create", "contract_take", "contract_work", "contract_cancel":
		r.contractAct(p, m, reply, fail, txn)

	case "trade_invite", "trade_accept", "trade_offer", "trade_lock", "trade_confirm", "trade_cancel":
		r.tradeAct(p, m, reply, fail)

	case "pk_bot", "pk_invite", "pk_accept", "pk_decline", "pk_tap", "pk_pick":
		r.pkAct(p, m, reply, fail)

	case "eat_item", "item_buy", "item_sell", "plant", "water", "harvest", "prefs", "land_buy", "land_info", "pet_ride", "weekly", "water_friend":
		r.logicAct(p, m, reply, fail, txn)

	case "bus_stops", "bus_board", "bus_alight", "xeom_accept", "xeom_decline", "xeom_call", "xeom_leave", "npc_choice":
		r.npcAct(p, m, reply, fail, txn)

	case "fish_start", "fish_cancel", "fish_finish":
		r.fishAct(p, m, reply, fail, txn)

	default:
		fail(errors.New("Hành động không hợp lệ."))
	}
}

// carrierOK: thao tác được phép khi đang ngồi xe NPC.
var carrierOK = map[string]bool{"bus_alight": true, "bus_stops": true, "xeom_leave": true, "npc_choice": true, "prefs": true, "eat_worm": true, "weekly": true,
	"eat_raw": true, "eat_food": true, "eat_item": true, "equip": true, "outfit_load": true, "outfit_save": true, "title": true, "show": true,
	"follow": true, "rename_pet": true, "care": true, "xeom_decline": true}

func containsInt(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

func restore(a *Account, snap []byte) {
	var fresh Account
	if json.Unmarshal(snap, &fresh) == nil {
		fresh.normalize()
		fresh.seq = a.seq
		*a = fresh
	}
}

func (r *Room) moveZone(p *Player, to string) {
	from := p.zone
	r.stand(p)
	if p.trade != nil {
		r.closeTrade(p.trade, "Một bên đã rời khu, giao dịch huỷ.")
	}
	if from == r.stationZone() && to != from {
		r.cancelTrip(p, "Bạn đã rời bến trước giờ xe chạy.")
	}
	r.abandonDig(p)
	r.cancelInvitesFor(p)
	r.broadcastZone(from, p, map[string]any{"t": "pleave", "pid": p.PID})
	p.zone = to
	r.discover(p, "zone:"+to)
	p.pos = r.freeSpawn(r.zones[to])
	p.ry = math.Pi
	p.in = Vec{}
	p.sess.Send(map[string]any{"t": "zone", "zone": to, "pos": p.pos, "ry": p.ry, "players": r.zonePlayers(to, p)})
	r.broadcastZone(to, p, map[string]any{"t": "pjoin", "p": r.publicOf(p)})
}

func (r *Room) broadcastLook(p *Player) {
	eq := map[string]string{}
	for k, v := range p.acc.Equipped {
		eq[k] = v
	}
	_, gname, gcolor := r.hub.store.Guilds.GuildOf(p.acc.ID)
	r.broadcastZone(p.zone, nil, map[string]any{"t": "plook", "pid": p.PID, "equipped": eq, "title": titleName(p.acc.Title), "level": p.acc.Level,
		"pet": petOf(p.acc), "guild": gname, "gcolor": gcolor})
}

// enterArea cập nhật khu theo vị trí sau mỗi bước đi trên bản đồ liền.
func (r *Room) enterArea(p *Player) {
	if r.world == nil {
		return
	}
	area := r.world.AreaAt(p.pos)
	if area == p.zone {
		return
	}
	if p.zone == r.stationZone() {
		r.cancelTrip(p, "Bạn đã rời bến trước giờ xe chạy.")
	}
	if p.match != nil && p.zone == "arena" {
		r.endMatchFor(p, "left") // chạy khỏi sân tỷ thí giữa trận coi như bỏ cuộc
	}
	p.zone = area
	r.discover(p, "zone:"+area)
	if area == "garden" && !p.acc.Tutorial["reach_garden"] {
		p.acc.step("reach_garden") // hướng dẫn: tự đi bộ hoặc đi xe tới Vườn giun
		p.dirty = true
	}
}
func (r *Room) stand(p *Player) {
	if p.seat == "" {
		return
	}
	p.seat = ""
	r.broadcastZone(p.zone, nil, map[string]any{"t": "nest", "pid": p.PID, "nest": ""})
}
