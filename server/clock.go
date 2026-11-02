package main

import (
	"hash/fnv"
	"math"
	"strconv"
	"time"
)

// Ngày/đêm và thời tiết (PLAN_LOGIC_GAME 16). Cả hai tính thuần từ giờ máy chủ nên mọi phòng, mọi người chơi
// thấy cùng một trời; client chỉ vẽ theo.

const (
	GameDay      = 48 * time.Minute // một ngày game: 24 phút ngày, 24 phút đêm
	RainSlot     = 6 * time.Minute  // thời tiết đổi theo từng khung 6 phút
	RainChance   = 10               // % số khung có mưa nhẹ (≈ 10% thời gian)
	NightGlow    = 1.5              // đêm: giun phát sáng ×1,5 tỉ lệ
	RainDigCut   = 0.7              // mưa: giun ngoi lên, hồi đất −30%
	RainXeOmFare = 1                // mưa: xe ôm phụ thu khăn lau
)

// NightWorms: loài phát sáng được ưu ái ban đêm.
var NightWorms = map[string]bool{"sang": true, "ngoc": true}

type Clock struct {
	Phase    float64 `json:"phase"` // 0 bình minh · 0,25 trưa · 0,5 hoàng hôn · 0,75 nửa đêm
	Night    bool    `json:"night"`
	Rain     bool    `json:"rain"`
	NextIn   float64 `json:"nextIn"`   // giây tới lúc đổi ngày ↔ đêm
	RainLeft float64 `json:"rainLeft"` // giây tới lúc đổi thời tiết
	Day      float64 `json:"day"`      // độ dài một ngày game (giây), client tự chạy đồng hồ giữa hai lần báo
}

func rainAt(slot int64) bool {
	h := fnv.New32a()
	h.Write([]byte("mua:" + strconv.FormatInt(slot, 10)))
	return h.Sum32()%100 < RainChance
}

func GameClock(now time.Time) Clock {
	day := GameDay.Seconds()
	t := math.Mod(float64(now.UnixMilli())/1000, day)
	c := Clock{Phase: t / day, Day: day}
	c.Night = c.Phase >= 0.5
	if c.Night {
		c.NextIn = day - t
	} else {
		c.NextIn = day/2 - t
	}
	slot := now.UnixMilli() / RainSlot.Milliseconds()
	c.Rain = rainAt(slot)
	end := slot + 1
	for rainAt(end) == c.Rain && end < slot+50 {
		end++
	}
	c.RainLeft = float64(end*RainSlot.Milliseconds()-now.UnixMilli()) / 1000
	return c
}

// glowFor: hệ số tỉ lệ của giun phát sáng lúc now.
func glowFor(now time.Time) float64 {
	if GameClock(now).Night {
		return NightGlow
	}
	return 1
}

// daypart cho lời chào theo buổi.
func daypart(now time.Time) string {
	switch p := GameClock(now).Phase; {
	case p < 0.2:
		return "buổi sáng"
	case p < 0.3:
		return "buổi trưa"
	case p < 0.5:
		return "buổi chiều"
	}
	return "buổi tối"
}

// clockTick báo cho cả phòng khi trời chuyển ngày/đêm hoặc đổi thời tiết.
func (r *Room) clockTick(now time.Time) {
	c := GameClock(now)
	if r.clockSent != nil && r.clockSent.Night == c.Night && r.clockSent.Rain == c.Rain {
		return
	}
	prev := r.clockSent
	r.clockSent = &c
	if prev == nil {
		return // welcome đã kèm đồng hồ
	}
	r.broadcastRoom(nil, map[string]any{"t": "clock", "clock": c})
	if c.Night != prev.Night {
		r.trafficForClock(c.Night, now)
	}
	switch {
	case c.Rain && !prev.Rain:
		r.system("🌧 Trời đổ mưa nhẹ: giun ngoi lên, ụ đất hồi nhanh hơn. Xe ôm phụ thu 1 thóc khăn lau.")
	case !c.Rain && prev.Rain:
		r.system("🌤 Tạnh mưa rồi.")
	case c.Night:
		r.system("🌙 Trời tối rồi: đèn lồng đã thắp, Giun phát sáng ngoi lên nhiều hơn.")
	default:
		r.system("🌅 Trời sáng! Gà gáy, làng lại nhộn nhịp.")
	}
}
