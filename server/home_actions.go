package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"
)

// homeAct xử lý thao tác trong nhà riêng. Chỉ chủ nhà được chỉnh sửa ở V1 (mục 9.1).
// Thay đổi chạm cả ví và nhà được lưu cùng nhau; lỗi lưu thì hoàn nguyên cả hai.
func (r *Room) homeAct(p *Player, m *inMsg, reply func(ActResult), fail func(error)) {
	if r.home == nil {
		fail(errors.New("Chỉ thao tác được trong chuồng riêng."))
		return
	}
	h := r.home
	if m.A == "party_cheer" {
		// Khách và chủ cùng ăn mừng ở bàn tiệc khi tiệc đang mở.
		if !h.Partying(time.Now()) {
			fail(errors.New("Chuồng này đang không mở tiệc."))
			return
		}
		near := false
		for _, o := range h.Objects {
			if o.Kind == "ban_tiec" && p.pos.Dist(Vec{float64(o.X), float64(o.Z)}) < 2.2 {
				near = true
			}
		}
		if !near || time.Since(p.emoteAt) < 3*time.Second {
			fail(errors.New("Lại gần bàn tiệc nhé."))
			return
		}
		p.emoteAt = time.Now()
		r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "party"})
		reply(ActResult{OK: true})
		return
	}
	if p.acc.ID != h.Owner {
		fail(errors.New("Chỉ chủ chuồng được chỉnh sửa."))
		return
	}
	now := time.Now()
	b := r.hub.bal
	zone := r.zones["home"]
	needCow := func() bool {
		if nearKind(zone, p.pos, "npc_cow") == nil {
			fail(errors.New("Gặp Bò Mộc để mua đồ, nâng đất hoặc thuê xây."))
			return false
		}
		return true
	}

	commit := func(apply func() (ActResult, error)) {
		accSnap, _ := json.Marshal(p.acc)
		homeSnap, _ := json.Marshal(h)
		tier := h.Tier
		res, err := apply()
		rollback := func() {
			restore(p.acc, accSnap)
			var fresh Home
			if json.Unmarshal(homeSnap, &fresh) == nil {
				*h = fresh
			}
		}
		if err != nil {
			rollback()
			fail(err)
			return
		}
		res.OK = true
		p.acc.Remember(m.RID, res)
		if err := r.hub.store.Homes.Save(h); err != nil {
			log.Printf("save home %s: %v", h.Owner, err)
			rollback()
			fail(errors.New("Máy chủ tạm không lưu được, thao tác chưa thực hiện."))
			return
		}
		if err := r.hub.store.Save(p.acc); err != nil {
			log.Printf("save %s: %v", p.acc.ID, err)
			rollback()
			_ = r.hub.store.Homes.Save(h)
			fail(errors.New("Máy chủ tạm không lưu được, thao tác chưa thực hiện."))
			return
		}
		if h.Tier != tier {
			// Đất rộng hơn: dựng lại khu và đưa mọi người về điểm an toàn trong ranh mới.
			r.zones["home"] = h.Zone()
			for _, o := range r.players {
				o.pos = r.zones["home"].Clamp(o.pos)
			}
		}
		r.broadcastHome()
		reply(res)
	}

	switch m.A {
	case "home_buy":
		if !needCow() {
			return
		}
		commit(func() (ActResult, error) {
			if err := h.BuyDecor(p.acc, m.ID, now); err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: "Đã mua " + decorByID(m.ID).Name + " — vào kho, đặt ở đâu tuỳ bạn."}, nil
		})
	case "home_place":
		commit(func() (ActResult, error) {
			o, err := h.Place(m.ID, int(math.Round(m.X)), int(math.Round(m.Z)), m.I)
			return ActResult{Data: map[string]any{"obj": o}}, err
		})
	case "home_move":
		commit(func() (ActResult, error) {
			return ActResult{}, h.Move(m.ID, int(math.Round(m.X)), int(math.Round(m.Z)), m.I)
		})
	case "home_pickup":
		if p.acc.Beds["home:"+m.ID] != nil {
			fail(errors.New("Luống đang có cây. Thu hoạch xong rồi hãy cất luống vào kho."))
			return
		}
		commit(func() (ActResult, error) { return ActResult{Msg: "Đã cất vào kho."}, h.Pickup(m.ID) })
	case "home_build":
		if !needCow() {
			return
		}
		if r.hub.store.Contracts.Active(h.Owner) != nil {
			fail(errors.New("Bạn đang có hợp đồng thuê thợ xây; huỷ hoặc chờ hợp đồng xong."))
			return
		}
		design, material := m.To, m.Mode
		if design == "" {
			design = "chuong_nho"
		}
		if material == "" {
			material = "go"
		}
		commit(func() (ActResult, error) {
			dur, err := h.BuildHouse(p.acc, design, material, m.ID, now, buildScale(b))
			if err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Bò Mộc bắt tay vào xây %s bằng %s — xong sau %s.", designByID(design).Name, MaterialNames[material], humanDur(dur))}, nil
		})
	case "home_upgrade_house":
		if !needCow() {
			return
		}
		commit(func() (ActResult, error) {
			dur, err := h.UpgradeHouse(p.acc, m.Mode, now, buildScale(b))
			if err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Bò Mộc nâng cấp nhà lên %s — xong sau %s.", MaterialNames[m.Mode], humanDur(dur))}, nil
		})
	case "home_access":
		if !contains(HomeAccess, m.ID) {
			fail(errors.New("Quyền vào không hợp lệ."))
			return
		}
		commit(func() (ActResult, error) {
			h.Access = m.ID
			return ActResult{}, nil
		})
	case "home_party":
		commit(func() (ActResult, error) {
			if err := h.OpenParty(m.I, now); err != nil {
				return ActResult{}, err
			}
			r.system(fmt.Sprintf("🎉 %s mở tiệc %d phút! Ai cũng ghé được — tối đa %d người.", h.OwnerName, m.I, h.capacity()))
			return ActResult{Msg: "Đã mở tiệc, chuồng hiện trong danh sách “đang mở tiệc”."}, nil
		})
	case "home_kick":
		t := r.byPID(m.Target)
		if t == nil || t == p {
			fail(errors.New("Không tìm thấy khách này."))
			return
		}
		t.sess.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Chủ chuồng đã mời bạn về sảnh."})
		go r.hub.LeaveHome(t.sess)
		reply(ActResult{OK: true, Msg: "Đã mời " + t.acc.Char.Name + " ra về."})
	}
}

func humanDur(d time.Duration) string {
	if d >= time.Minute {
		return fmt.Sprintf("%d phút", int(d.Minutes()))
	}
	return fmt.Sprintf("%d giây", int(d.Seconds()))
}
