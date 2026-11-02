package main

import (
	"testing"
	"time"
)

// Đạo diễn làng sống giữ đúng hạn mức của PLAN_LOGIC_GAME 5.2: trêu ≤ 1 lần/5 phút và ≤ 6 lần/giờ, không trêu
// 5 phút đầu, cameo ≤ 1 lần/8 phút; tắt trêu thì không bị trêu; mức "Ít" thì không có cameo; toàn phòng
// ≤ 3 câu tán gẫu mỗi giây.

// simulateCameos chạy riêng cameoTick của người chơi trong 2 giờ giả lập (2 nhịp/giây) và ghi lại lúc bị trêu
// và lúc có cameo. Diễn viên cameo được gỡ ngay để không chặn lượt sau bằng giới hạn số cameo đang chạy.
func simulateCameos(t *testing.T, e *env, user string, prefs Prefs) (start time.Time, teases, cameos []time.Time) {
	room := e.roomOf(t)
	room.DoWait(func() {
		var p *Player
		for _, x := range room.players {
			if x.acc.Username == user {
				p = x
			}
		}
		if p == nil {
			t.Error("không thấy người chơi trong phòng")
			return
		}
		p.acc.Prefs = prefs
		start = time.Now()
		p.joinedAt, p.moving = start, true
		p.memo = npcMemo{greetAt: map[string]time.Time{}}
		for i := 0; i < 2*3600*2; i++ {
			now := start.Add(time.Duration(i) * 500 * time.Millisecond)
			tz, cz := p.memo.teaseAt, p.memo.cameoAt
			room.cameoTick(p, now)
			if p.memo.teaseAt != tz {
				teases = append(teases, now)
			}
			if p.memo.cameoAt != cz {
				cameos = append(cameos, now)
			}
			for _, a := range room.actors {
				if a.Kind == KindCameo {
					room.dropActor(a)
				}
			}
		}
		p.memo = npcMemo{greetAt: map[string]time.Time{}, teaseAt: time.Now(), cameoAt: time.Now()}
	})
	return
}

func TestDirectorTeaseAndCameoLimits(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "dibo", "Gà Đi Bộ", func(acc *Account) { acc.Level = 3; acc.Mana = 100 })
	e.warp(a, "village")

	start, teases, cameos := simulateCameos(t, e, "dibo", Prefs{})
	if len(teases) == 0 {
		t.Fatal("2 giờ đi bộ ngoài đường phải gặp ít nhất một lần thiếu gia hoặc hai mẹ con")
	}
	if teases[0].Sub(start) <= 5*time.Minute {
		t.Fatalf("không trêu người mới trong 5 phút đầu: lần đầu sau %v", teases[0].Sub(start))
	}
	for i := range teases {
		if i > 0 && teases[i].Sub(teases[i-1]) <= 5*time.Minute {
			t.Fatalf("trêu dày quá: %v", teases[i].Sub(teases[i-1]))
		}
		n := 0
		for _, u := range teases[:i+1] {
			if teases[i].Sub(u) < time.Hour {
				n++
			}
		}
		if n > TeasePerHour {
			t.Fatalf("%d lần trêu trong một giờ, tối đa %d", n, TeasePerHour)
		}
	}
	for i := 1; i < len(cameos); i++ {
		if cameos[i].Sub(cameos[i-1]) <= 8*time.Minute {
			t.Fatalf("cameo dày quá: %v", cameos[i].Sub(cameos[i-1]))
		}
	}

	if _, teases, _ := simulateCameos(t, e, "dibo", Prefs{NoTease: true}); len(teases) != 0 {
		t.Fatalf("đã tắt trêu mà vẫn bị trêu %d lần", len(teases))
	}
	if _, _, cameos := simulateCameos(t, e, "dibo", Prefs{Chatter: "low"}); len(cameos) != 0 {
		t.Fatalf("mức Ít không có cameo, có %d", len(cameos))
	}
}

func TestRoomSayBudget(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "nghe", "Gà Nghe", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "village")
	room := e.roomOf(t)
	room.DoWait(func() {
		var p *Player
		for _, x := range room.players {
			p = x
		}
		room.sayWin = time.Time{}
		for i := 0; i < 10; i++ {
			room.say("tan_gau", "Gà Qua Đường", "Hôm nay trời đẹp ghê!", p.pos, nil)
		}
		room.say("ga_cu_cuc", "Cụ Cục", "Cháu ơi, lại đây ông bảo!", p.pos, p)
	})
	chatter, direct := 0, 0
	for {
		m, ok := a.tryExpect("say", 400*time.Millisecond)
		if !ok {
			break
		}
		switch m["id"] {
		case "tan_gau":
			chatter++
		case "ga_cu_cuc":
			direct++
		}
	}
	if chatter != SayPerSec || direct != 1 {
		t.Fatalf("mỗi giây tối đa %d câu tán gẫu, câu nói riêng luôn tới: %d tán gẫu, %d riêng", SayPerSec, chatter, direct)
	}
}

// tryExpect như expect nhưng không đánh trượt test khi hết giờ.
func (c *client) tryExpect(typ string, within time.Duration) (msg, bool) {
	for i, m := range c.backlog {
		if m["t"] == typ {
			c.backlog = append(c.backlog[:i], c.backlog[i+1:]...)
			return m, true
		}
	}
	deadline := time.After(within)
	for {
		select {
		case m, ok := <-c.in:
			if !ok {
				return nil, false
			}
			if m["t"] == typ {
				return m, true
			}
		case <-deadline:
			return nil, false
		}
	}
}

// Nghỉ ổ cạnh bạn bè (PLAN_LOGIC_GAME 16): đủ một phút thì bạn đồng hành đang đi theo +1 thân thiết; người lạ
// hoặc ngồi xa thì không.
func TestNestBuddyAffinity(t *testing.T) {
	e := newEnv(t, t.TempDir())
	pet := func(acc *Account) {
		acc.Companions = []Companion{{ID: "c1", Name: "Bé", Species: "ga_con"}}
		acc.Following = "c1"
	}
	a, aid := e.seeded(t, "ngoi1", "Gà Ngồi Một", pet)
	b, bid := e.seeded(t, "ngoi2", "Gà Ngồi Hai", pet)
	_, _ = a, b
	room := e.roomOf(t)
	affinity := func(friends bool, gap float64) (int, int) {
		var pa, pb *Player
		room.DoWait(func() {
			for _, x := range room.players {
				switch x.acc.ID {
				case aid:
					pa = x
				case bid:
					pb = x
				}
			}
			pa.seat, pb.seat = "o1", "o2"
			pb.pos = Vec{pa.pos.X + gap, pa.pos.Z}
			pa.memo.buddySince, pb.memo.buddySince = time.Time{}, time.Time{}
			pa.acc.Companions[0].Affinity, pb.acc.Companions[0].Affinity = 0, 0
			now := time.Now()
			for s := 0; s <= 125; s++ {
				at := now.Add(time.Duration(s) * time.Second)
				room.nestBuddy(pa, at)
				room.nestBuddy(pb, at)
			}
			pa.seat, pb.seat = "", ""
		})
		return pa.acc.Companions[0].Affinity, pb.acc.Companions[0].Affinity
	}
	if x, y := affinity(false, 1.5); x != 0 || y != 0 {
		t.Fatalf("chưa là bạn thì không cộng: %d %d", x, y)
	}
	if _, err := e.store.Social.Request(aid, bid); err != nil {
		t.Fatal(err)
	}
	if err := e.store.Social.Respond(bid, aid, true); err != nil {
		t.Fatal(err)
	}
	if x, y := affinity(true, 1.5); x != 2 || y != 2 {
		t.Fatalf("ngồi cạnh bạn 2 phút: +2 thân thiết mỗi bên, được %d %d", x, y)
	}
	if x, _ := affinity(true, 6); x != 0 {
		t.Fatal("ngồi xa nhau thì không tính")
	}
}
