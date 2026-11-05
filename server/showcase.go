package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Lô trưng bày ở Xóm Rơm Vàng (PLAN_LOGIC_GAME 13.2, P5b): 8 lô dọc đường xóm hiện nhà của người chơi đang
// online trong kênh, ưu tiên nhà mình rồi tới bạn bè. Thiếu người thì lô hiện nhà mẫu của Chủ đầu tư. Nhà vẫn
// là phòng riêng; lô chỉ là "mặt tiền" để ghé thăm.

// ShowcaseLots: tâm nhà (X, Z) và cổng (GX, GZ) của 8 lô, toạ độ trong khu Xóm Rơm Vàng; client vẽ theo cùng
// bảng này (scenery.js neighborhood).
var ShowcaseLots = []struct{ X, Z, GX, GZ float64 }{
	{-13, -12, -5.2, -10.4}, {13, -12, 5.2, -10.4}, {-13, 4, -5.2, 5.6}, {13, 4, 5.2, 5.6},
	{-12, -3.8, -5.4, -3.8}, {12, -3.8, 5.4, -3.8}, {-12, 12, -5.4, 12}, {-12, -19.4, -5.4, -19.4},
}

type ShowLot struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	Design   string `json:"design"`
	Material string `json:"material"`
	Roof     string `json:"roof"`
	State    string `json:"state"` // building | done
	Mine     bool   `json:"mine,omitempty"`
	Friend   bool   `json:"friend,omitempty"`
}

func showcaseItems() []Item {
	out := make([]Item, len(ShowcaseLots))
	for i, l := range ShowcaseLots {
		out[i] = Item{ID: fmt.Sprintf("lo_%d", i+1), Kind: "showcase_lot", Label: fmt.Sprintf("Lô %d", i+1), Emo: "🏡", X: l.GX, Z: l.GZ, R: 2.2}
	}
	return out
}

// showcaseFor xếp nhà lên lô cho một người xem: nhà mình, rồi bạn bè, rồi người khác theo thứ tự vào kênh.
func (r *Room) showcaseFor(v *Player, houses map[string]HomeIndexEntry, order []*Player, now time.Time) []*ShowLot {
	type cand struct {
		p    *Player
		rank int
	}
	var list []cand
	for i, q := range order {
		if _, ok := houses[q.acc.ID]; !ok {
			continue
		}
		rank := 2
		switch {
		case q == v:
			rank = 0
		case r.hub.store.Social.AreFriends(v.acc.ID, q.acc.ID):
			rank = 1
		}
		list = append(list, cand{q, rank*1000 + i})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].rank < list[j].rank })
	lots := make([]*ShowLot, len(ShowcaseLots))
	for i := 0; i < len(lots) && i < len(list); i++ {
		q, e := list[i].p, houses[list[i].p.acc.ID]
		state := e.State
		if state == "building" && !now.Before(e.DoneAt) {
			state = "done"
		}
		lots[i] = &ShowLot{Owner: q.acc.ID, Name: q.acc.Char.Name, Design: e.Design, Material: e.Material, Roof: e.Roof,
			State: state, Mine: q == v, Friend: list[i].rank/1000 == 1}
	}
	return lots
}

// syncShowcase gửi lô trưng bày cho người trong kênh khi có thay đổi (mỗi 5 giây).
func (r *Room) syncShowcase(now time.Time) {
	if r.zones["neighborhood"] == nil || r.Kind != "public" {
		return
	}
	order := make([]*Player, 0, len(r.players))
	ids := make([]string, 0, len(r.players))
	for _, p := range r.players {
		if p.acc.Char != nil && p.goneAt.IsZero() {
			order = append(order, p)
			ids = append(ids, p.acc.ID)
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i].joinedAt.Before(order[j].joinedAt) })
	houses := r.hub.store.Homes.Houses(ids)
	for _, v := range order {
		lots := r.showcaseFor(v, houses, order, now)
		b, _ := json.Marshal(lots)
		if string(b) == v.showSent {
			continue
		}
		v.showSent = string(b)
		v.sess.Send(map[string]any{"t": "showcase", "lots": lots})
	}
}
