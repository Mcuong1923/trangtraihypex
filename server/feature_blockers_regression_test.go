package main

import (
	"testing"
	"time"
)

// Test hồi quy cho R1 và D1 (lượt rà 2) trong plans/reports/code-reviewer-261105-1412-game-logic-review.md.
// B1–B3 là lỗi client (vẽ lại bảng, nút xuống xe), kiểm bằng tay trên trình duyệt.

// R1: "Bán hết" từng gửi một lệnh mỗi con và bị giới hạn tốc độ message chặn khoảng con thứ 41.
// Giờ gửi một lệnh kèm số lượng; máy chủ bán trong một giao dịch.
func TestSellAllInOneMessage(t *testing.T) {
	const have = 55 // vượt MsgBurst (40) để chắc chắn cách cũ sẽ bị chặn
	w := &Worms[0]
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "banhet", "Gà Bán Hết", func(acc *Account) { acc.Worms[w.ID] = have })
	e.warp(a, "market")
	a.walkToItem("market", "sap_giun")

	r := a.mustOK("sell", msg{"id": w.ID, "i": have + 10}) // xin nhiều hơn số có: bán hết số đang có
	data := r["data"].(map[string]any)
	if data["sold"].(float64) != have {
		t.Fatalf("phải bán %d con, bán %v", have, data["sold"])
	}
	if want := float64(SellPrice(nil, w, time.Now()) * have); data["got"].(float64) != want {
		t.Fatalf("tổng thóc %v, muốn %v", data["got"], want)
	}
	if n := a.count("worms", w.ID); n != 0 {
		t.Fatalf("túi còn %v con", n)
	}
	if r := a.act("sell", msg{"id": w.ID, "i": 5}); r["ok"] == true {
		t.Fatal("túi rỗng thì không bán được")
	}
}

// R1: bán lẻ (không gửi số lượng) vẫn chỉ bán một con; số âm không bán ngược.
func TestSellWormsCounts(t *testing.T) {
	w := &Worms[0]
	now := time.Now()
	a := NewAccount("aaaaaaaaaaaaaaaaaaaaaaaa", "dem", now)
	a.Worms[w.ID] = 3
	for _, n := range []int{0, -4} {
		if _, sold, _, err := a.SellWorms(w.ID, n, now); err != nil || sold != 1 {
			t.Fatalf("n=%d: bán %d con, lỗi %v", n, sold, err)
		}
	}
	if got, sold, _, err := a.SellWorms(w.ID, 9, now); err != nil || sold != 1 || got != SellPrice(a, w, now) {
		t.Fatalf("còn 1 con: bán %d, nhận %d, lỗi %v", sold, got, err)
	}
	if a.Worms[w.ID] != 0 || a.Stats.Sold != 3 {
		t.Fatalf("còn %d con, đếm đã bán %d", a.Worms[w.ID], a.Stats.Sold)
	}
}

// expireCoinPicks đẩy hạn chọn của mọi lượt Úp ngửa đang chờ về quá khứ (khỏi chờ đủ CoinPickTimeout).
func (e *env) expireCoinPicks() {
	e.hub.mu.Lock()
	rooms := append([]*Room{}, e.hub.rooms...)
	e.hub.mu.Unlock()
	for _, r := range rooms {
		r.DoWait(func() {
			for _, mt := range r.matches {
				if mt.Waiting {
					mt.PickBy = time.Now().Add(-time.Second)
				}
			}
		})
	}
}

// D1: người tới lượt chọn mặt không bấm thì trận Úp ngửa từng treo vô hạn; giờ quá hạn máy chủ chọn thay
// và trận kết thúc bình thường (không ai bị xử thua vì bỏ trận).
func TestCoinPickTimeoutAutoPicks(t *testing.T) {
	if CoinPickTimeout <= 0 || CoinPickTimeout > 30*time.Second {
		t.Fatalf("hạn chọn Úp ngửa bất thường: %v", CoinPickTimeout)
	}
	e := newEnv(t, t.TempDir())
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	b, _ := e.playerTok(t, "bob", "Gà Mơ")
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
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "coin"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})
	st := a.expect("pk_start", 2*time.Second)

	for round := 0; round < CoinRounds; round++ {
		rd := a.expect("pk_round", 5*time.Second)
		if left := rd["pickLeft"].(float64); left <= 0 || left > CoinPickTimeout.Seconds() {
			t.Fatalf("lượt %d: hạn chọn gửi client %v giây", round, left)
		}
		// Không ai bấm chọn: hết hạn thì máy chủ phải tự tung.
		e.expireCoinPicks()
		f := a.expect("pk_flip", 2*time.Second)
		if f["round"].(float64) != float64(round) {
			t.Fatalf("lật nhầm lượt: %v", f["round"])
		}
	}
	end := b.expect("pk_end", 6*time.Second)
	if end["id"] != st["id"] || end["reason"] != "" || end["winner"].(float64) < 0 {
		t.Fatalf("trận phải kết thúc bình thường: %v", end)
	}
}

// D1: lượt của Gà Máy giờ chạy theo hạn chọn trong tick (bỏ goroutine ngủ); máy phải tự lật sau CoinBotPickDelay.
// Lệnh chọn tới sau khi lượt đã lật thì bị từ chối, không lật hai lần.
func TestCoinBotPicksAndLatePickRejected(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "dauomay", "Gà Đấu Máy", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "arena")
	a.walkToItem("arena", "cho_muc")
	a.mustOK("pk_bot", msg{"mode": "coin"})
	st := a.expect("pk_start", 2*time.Second)
	for round := 0; round < CoinRounds; round++ {
		rd := a.expect("pk_round", 5*time.Second)
		if rd["chooser"].(float64) == st["you"].(float64) {
			// Lượt của người: để hết hạn rồi bấm trễ.
			e.expireCoinPicks()
			a.expect("pk_flip", 2*time.Second)
			if r := a.act("pk_pick", msg{"id": st["id"], "side": "up"}); r["ok"] == true {
				t.Fatal("chọn sau khi lượt đã lật phải bị từ chối")
			}
			continue
		}
		start := time.Now()
		a.expect("pk_flip", 3*time.Second) // Gà Máy tự chọn, không ai đẩy hạn
		if d := time.Since(start); d < CoinBotPickDelay/2 {
			t.Fatalf("Gà Máy lật quá sớm: %v", d)
		}
	}
	if end := a.expect("pk_end", 6*time.Second); end["id"] != st["id"] || end["reason"] != "" {
		t.Fatalf("trận với Gà Máy phải kết thúc bình thường: %v", end)
	}
}
