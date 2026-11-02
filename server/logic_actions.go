package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Hành động của PLAN_LOGIC_GAME: kho đồ, tiệm vật liệu/hạt giống, trồng cây, điểm danh, cài đặt NPC.
func (r *Room) logicAct(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn func(func() (ActResult, error))) {
	now := time.Now()
	zone := r.zones[p.zone]
	switch m.A {
	case "eat_item":
		txn(func() (ActResult, error) {
			gain, err := p.acc.EatItem(m.ID)
			if err != nil {
				return ActResult{}, err
			}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			return ActResult{Msg: fmt.Sprintf("Ăn %s: +%d độ no", itemByID(m.ID).Name, gain)}, nil
		})

	case "item_buy":
		shop := ""
		switch {
		case nearKind(zone, p.pos, "npc_cow") != nil || nearKind(zone, p.pos, "material_shop") != nil:
			shop = "vatlieu"
		case nearKind(zone, p.pos, "shop_food") != nil:
			shop = "thucan"
		default:
			fail(errors.New("Vật liệu bán ở tiệm của Bò Mộc (Xóm Rơm Vàng); hạt giống bán ở Quầy thức ăn."))
			return
		}
		n := max(1, m.I)
		txn(func() (ActResult, error) {
			if err := p.acc.BuyItem(m.ID, n, shop, now); err != nil {
				return ActResult{}, err
			}
			it := itemByID(m.ID)
			return ActResult{Msg: fmt.Sprintf("Đã mua %d %s (−%d thóc)", n, it.Name, it.Buy*n)}, nil
		})

	case "item_sell":
		if nearKind(zone, p.pos, "shop_food") == nil {
			fail(errors.New("Bán nông sản ở Quầy thức ăn tại Chợ thóc."))
			return
		}
		n := max(1, m.I)
		txn(func() (ActResult, error) {
			got, err := p.acc.SellItem(m.ID, n, now)
			if err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Bán %d %s: +%d thóc", n, itemByID(m.ID).Name, got)}, nil
		})

	case "plant", "water", "harvest":
		bed, err := r.bedKey(p, m.ID)
		if err != nil {
			fail(err)
			return
		}
		txn(func() (ActResult, error) {
			switch m.A {
			case "plant":
				pl, err := p.acc.PlantSeed(bed, m.Mode, m.Eat, now, r.growScale())
				if err != nil {
					return ActResult{}, err
				}
				r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "dig"})
				return ActResult{Msg: fmt.Sprintf("Đã gieo %s — chín sau %s.", itemByID(pl.Seed).Name, humanDur(pl.Ready.Sub(now)))}, nil
			case "water":
				if err := p.acc.WaterBed(bed, now); err != nil {
					return ActResult{}, err
				}
				return ActResult{Msg: "Đã tưới — cây lớn nhanh hơn một chút."}, nil
			}
			crop, n, err := p.acc.Harvest(bed, now)
			if err != nil {
				return ActResult{}, err
			}
			if crop == "phan_u" {
				return ActResult{Msg: "Cây héo mất rồi, chỉ còn 1 phân ủ (bón cho lần trồng sau: +1 nông sản)."}, nil
			}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "eat"})
			return ActResult{Msg: fmt.Sprintf("Thu hoạch %d %s!", n, itemByID(crop).Name)}, nil
		})

	case "land_info":
		// Bảng lô đất của Ông Chủ Đầu Tư: lô, vị trí, giá; đọc nhà qua withHome để không đụng goroutine khác.
		var tier int
		var place string
		if err := r.hub.withHome(p.acc.ID, p.acc.Char.Name, func(h *Home) error { tier, place = h.Tier, h.LotPlace; return nil }); err != nil {
			fail(err)
			return
		}
		reply(ActResult{OK: true, Data: map[string]any{"tier": tier, "place": place, "tiers": LandTiers, "places": LandPlaces, "designs": HouseDesigns}})

	case "land_buy":
		if nearKind(zone, p.pos, "npc_investor") == nil {
			fail(errors.New("Mua đất ở Văn phòng đất nền của Ông Chủ Đầu Tư, cạnh cổng Xóm Rơm Vàng."))
			return
		}
		txn(func() (ActResult, error) {
			var moved, cost int
			err := r.hub.withHome(p.acc.ID, p.acc.Char.Name, func(h *Home) error {
				var e error
				moved, cost, e = h.BuyLot(p.acc, m.I, m.Mode, now)
				return e
			})
			if err != nil {
				return ActResult{}, err
			}
			msg := fmt.Sprintf("Đã mua %s · %s (−%d thóc). Vào nhà riêng để xem đất mới!", LandTiers[m.I].Name, placeByID(m.Mode).Name, cost)
			if moved > 0 {
				msg += fmt.Sprintf(" %d món đồ không còn chỗ hợp lệ đã về kho.", moved)
			}
			return ActResult{Msg: msg}, nil
		})

	case "pet_ride":
		switch {
		case !canRideGoose(p.acc):
			fail(errors.New("Cần dắt theo Ngỗng con ở mức Thân thiết (thân thiết 150+)."))
		case now.Before(p.gooseReady):
			fail(fmt.Errorf("Ngỗng đang nghỉ, cưỡi lại sau %s.", humanDur(p.gooseReady.Sub(now))))
		case p.ride != nil || p.rideOf != nil || p.carrier != nil:
			fail(errors.New("Xuống xe trước đã."))
		default:
			p.gooseUntil, p.gooseReady = now.Add(GooseRide), now.Add(GooseCooldown)
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "goose", "sec": GooseRide.Seconds()})
			reply(ActResult{OK: true, Msg: "🪿 Ngỗng cõng bạn chạy vèo vèo trong 20 giây!"})
		}

	case "weekly":
		reply(ActResult{OK: true, Data: r.hub.weekly.View(now)})

	case "prefs":
		chat := m.Mode
		if !contains([]string{"", "full", "normal", "low"}, chat) {
			fail(errors.New("Mức trò chuyện không hợp lệ."))
			return
		}
		txn(func() (ActResult, error) {
			p.acc.Prefs = Prefs{Chatter: chat, NoTease: !m.Eat}
			return ActResult{Msg: "Đã lưu cài đặt NPC."}, nil
		})

	default:
		fail(errors.New("Hành động không hợp lệ."))
	}
}

// bedKey kiểm tra người chơi đứng cạnh luống: luống cộng đồng ở Ao và vườn ("park:luong_N") hoặc luống rau
// trong nhà riêng của chính mình ("home:<mã đồ>").
func (r *Room) bedKey(p *Player, id string) (string, error) {
	zone := r.zones[p.zone]
	if strings.HasPrefix(id, "home:") {
		if r.home == nil || r.Owner != p.acc.ID {
			return "", errors.New("Chỉ chăm luống rau trong nhà của chính mình.")
		}
		oid := strings.TrimPrefix(id, "home:")
		for _, o := range r.home.Objects {
			if o.ID == oid && (o.Kind == "luong_rau" || o.Kind == "luong_ca_rot") {
				if p.pos.Dist(Vec{float64(o.X), float64(o.Z)}) > 2.4 {
					return "", errors.New("Lại gần luống rau hơn.")
				}
				return id, nil
			}
		}
		return "", errors.New("Không tìm thấy luống rau này.")
	}
	it := zone.Item(id)
	if it == nil || it.Kind != "bed" || p.pos.Dist(it.Pos()) > it.R+InteractSlack {
		return "", errors.New("Lại gần một luống ở Ao và vườn.")
	}
	return "park:" + id, nil
}

// growScale: bộ demo cho cây lớn nhanh gấp 10 lần để thử.
func (r *Room) growScale() float64 {
	if r.hub.bal.Name == "demo" {
		return 0.1
	}
	if r.hub.bal.Name == "test" {
		return 0.001
	}
	return 1
}
