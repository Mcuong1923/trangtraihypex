package main

import (
	"testing"
	"time"
)

// Bảng điểm Sân tỷ thí: khán giả (không đấu) nhận tên, tỉ số, kết quả; người vào sau thấy kết quả đang giữ;
// hết thời gian giữ thì bảng về trống.
func TestPKBoardSpectatorSeesScoreAndResult(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	b, _ := e.playerTok(t, "bob", "Gà Mơ")
	c, _ := e.playerTok(t, "carol", "Gà Xem")
	e.warp(a, "arena")
	e.warp(b, "arena")
	var bPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Mơ" {
			bPID = p["pid"].(string)
			break
		}
	}
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "peck"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})

	start := c.expect("pk_board", 2*time.Second)
	names, _ := start["names"].([]any)
	if start["state"] != "countdown" || len(names) != 2 || names[0] != "Gà Bông" || names[1] != "Gà Mơ" {
		t.Fatalf("bảng phải lên trận mới với tên hai bên: %v", start)
	}
	a.expect("pk_go", 6*time.Second)
	for i := 0; i < 3; i++ {
		a.send(msg{"t": "act", "a": "pk_tap", "id": start["id"]})
		time.Sleep(80 * time.Millisecond)
	}
	c.expectWhere("pk_board", 3*time.Second, func(m msg) bool { return m["state"] == "play" && m["a"] == float64(3) })

	r := e.roomOf(t)
	r.DoWait(func() {
		for _, q := range r.players {
			if q.PID == bPID {
				r.endMatchFor(q, "left")
			}
		}
	})
	end := c.expectWhere("pk_board", 3*time.Second, func(m msg) bool { return m["state"] == "end" })
	if end["winner"] != float64(0) || end["a"] != float64(3) || end["forfeit"] != true {
		t.Fatalf("bảng phải hiện kết quả: %v", end)
	}

	d, _ := e.playerTok(t, "dave", "Gà Đến Sau")
	if late := d.expect("pk_board", 2*time.Second); late["state"] != "end" || late["id"] != start["id"] {
		t.Fatalf("người vào lúc đang giữ kết quả phải thấy kết quả: %v", late)
	}

	r.DoWait(func() { r.pkb.holdUntil = time.Now().Add(-time.Second) })
	c.expectWhere("pk_board", 2*time.Second, func(m msg) bool { return m["state"] == "idle" })
}

// Nhiều trận cùng lúc: trận mới nhất lên bảng, trận cũ xong không đổi bảng; hết giữ kết quả thì bảng
// chuyển sang trận còn đang đấu; huỷ lúc đếm ngược hiện "huỷ", không có người thắng.
func TestPKBoardOverlappingMatchesAndCancel(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	b, _ := e.playerTok(t, "bob", "Gà Mơ")
	e.warp(a, "arena")
	e.warp(b, "arena")
	r := e.roomOf(t)
	r.DoWait(func() {
		var pa, pb *Player
		for _, q := range r.players {
			switch q.acc.Char.Name {
			case "Gà Bông":
				pa = q
			case "Gà Mơ":
				pb = q
			}
		}
		now := time.Now()
		r.startMatch(pa, nil, "coin", now)
		m1 := pa.match
		r.startMatch(pb, nil, "coin", now.Add(time.Millisecond))
		m2 := pb.match
		if r.pkb.id != m2.ID {
			t.Errorf("trận mới nhất phải lên bảng")
		}
		r.finishMatch(m2, "")
		if r.pkb.last["state"] != "end" || r.pkb.last["id"] != m2.ID {
			t.Errorf("trận trên bảng xong phải hiện kết quả: %v", r.pkb.last)
		}
		r.stepPKBoard(now.Add(pkBoardHold + time.Second))
		if r.pkb.id != m1.ID || r.pkb.last["id"] != m1.ID {
			t.Errorf("hết giữ kết quả phải chuyển sang trận còn đấu: %v", r.pkb.last)
		}
		r.finishMatch(m1, pa.PID) // bỏ đi lúc đang đếm ngược: huỷ trận
		if l := r.pkb.last; l["state"] != "end" || l["cancelled"] != true || l["winner"] != -1 || l["forfeit"] != false {
			t.Errorf("huỷ lúc đếm ngược: %v", l)
		}
		r.startMatch(pb, nil, "peck", now.Add(2*time.Millisecond))
		m3 := pb.match
		r.startMatch(pa, nil, "peck", now.Add(3*time.Millisecond))
		m4 := pa.match
		r.finishMatch(m3, "")
		if r.pkb.id != m4.ID || r.pkb.last["id"] != m4.ID || r.pkb.last["state"] == "end" {
			t.Errorf("trận cũ xong không được đổi bảng của trận mới: %v", r.pkb.last)
		}
		r.finishMatch(m4, "")
	})
}
