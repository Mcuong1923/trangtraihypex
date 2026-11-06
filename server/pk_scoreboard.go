package main

import (
	"math"
	"time"
)

// Bảng điểm 3D ở Sân tỷ thí (PLAN_GIAO_DIEN_BO_2 B02-04, QA-06): khán giả thấy cùng tỉ số và kết quả với
// hai đấu sĩ. Bảng chỉ hiện một trận: trận mới bắt đầu luôn được lên bảng; trận trên bảng kết thúc thì
// giữ kết quả pkBoardHold rồi chuyển sang trận khác còn đang đấu, không còn thì về bảng trống.

const (
	pkBoardGap  = 200 * time.Millisecond // gộp cập nhật tỉ số: tối đa 5 gói mỗi giây cho cả phòng
	pkBoardHold = 5 * time.Second        // thời gian giữ kết quả trận vừa xong trên bảng
)

type pkBoard struct {
	id        string         // trận đang lên bảng ("" = không có trận nào)
	dirty     bool           // trận trên bảng đổi tỉ số/trạng thái, chưa gửi
	sentAt    time.Time      // lần gửi gần nhất
	holdUntil time.Time      // đang giữ kết quả trận vừa xong
	last      map[string]any // gói gần nhất, gửi lại cho người mới vào phòng
}

// boardMsg dựng trạng thái trận cho bảng; "left" là số giây còn lại của đếm ngược hoặc hiệp Mổ nhanh.
func (r *Room) boardMsg(mt *Match, now time.Time) map[string]any {
	na, nb := mt.names()
	left := 0.0
	switch {
	case mt.State == "countdown":
		left = mt.CountEnd.Sub(now).Seconds()
	case mt.Mode == "peck":
		left = mt.PlayEnd.Sub(now).Seconds()
	}
	return map[string]any{"t": "pk_board", "id": mt.ID, "mode": mt.Mode, "modeName": PKModes[mt.Mode],
		"names": []string{na, nb}, "a": mt.ScoreA, "b": mt.ScoreB, "state": mt.State, "left": math.Round(max(0, left)*10) / 10,
		"round": mt.Round, "rounds": CoinRounds, "winner": -1}
}

func (r *Room) pkBoardSend(msg map[string]any, now time.Time) {
	r.pkb.last, r.pkb.sentAt, r.pkb.dirty = msg, now, false
	r.broadcastZone("arena", nil, msg)
}

// pkBoardStart: trận mới bắt đầu luôn thay trận cũ trên bảng (kể cả đang giữ kết quả).
func (r *Room) pkBoardStart(mt *Match, now time.Time) {
	r.pkb.id, r.pkb.holdUntil = mt.ID, time.Time{}
	r.pkBoardSend(r.boardMsg(mt, now), now)
}

// pkBoardTouch đánh dấu trận trên bảng vừa đổi; stepPKBoard gửi gộp.
func (r *Room) pkBoardTouch(mt *Match) {
	if mt.ID == r.pkb.id {
		r.pkb.dirty = true
	}
}

// pkBoardEnd gửi ngay kết quả của trận đang lên bảng và giữ nó pkBoardHold.
// forfeit: thắng vì đối thủ bỏ trận (bảng ghi "đối thủ bỏ trận" thay cho "KO!").
func (r *Room) pkBoardEnd(mt *Match, winner int, cancelled, forfeit bool, now time.Time) {
	if mt.ID != r.pkb.id {
		return
	}
	msg := r.boardMsg(mt, now)
	msg["state"], msg["winner"], msg["cancelled"], msg["forfeit"], msg["left"] = "end", winner, cancelled, forfeit, 0
	r.pkb.id, r.pkb.holdUntil = "", now.Add(pkBoardHold)
	r.pkBoardSend(msg, now)
}

// stepPKBoard chạy mỗi tick sau stepPK.
func (r *Room) stepPKBoard(now time.Time) {
	b := &r.pkb
	if b.id != "" {
		mt := r.matches[b.id]
		if mt != nil {
			if b.dirty && now.Sub(b.sentAt) >= pkBoardGap {
				r.pkBoardSend(r.boardMsg(mt, now), now)
			}
			return
		}
		b.id = "" // phòng hờ trận biến mất mà không qua finishMatch: không để bảng đứng mãi
	}
	if b.last == nil || now.Before(b.holdUntil) {
		return
	}
	if mt := r.newestMatch(); mt != nil {
		b.id = mt.ID
		r.pkBoardSend(r.boardMsg(mt, now), now)
	} else if b.last["state"] != "idle" {
		r.pkBoardSend(map[string]any{"t": "pk_board", "state": "idle"}, now)
	}
}

// newestMatch: trận bắt đầu gần nhất còn đang diễn ra.
func (r *Room) newestMatch() *Match {
	var best *Match
	for _, mt := range r.matches {
		if best == nil || mt.CountEnd.After(best.CountEnd) {
			best = mt
		}
	}
	return best
}

// sendPKBoard gửi trạng thái bảng hiện tại cho người vừa vào phòng hoặc vừa bước vào sân.
func (r *Room) sendPKBoard(p *Player) {
	if mt := r.matches[r.pkb.id]; mt != nil {
		p.sess.Send(r.boardMsg(mt, time.Now())) // dựng lại để số giây còn lại đúng lúc vào
		return
	}
	if r.pkb.last != nil && r.pkb.last["state"] != "idle" {
		p.sess.Send(r.pkb.last)
	}
}
