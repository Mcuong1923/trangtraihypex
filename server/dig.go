package main

import (
	"math"
	"time"
)

// Đào giun v2 (PLAN_LOGIC_GAME mục 4): mổ đất → giun ló → giằng co. Máy chủ gieo loài giun và seed lúc
// bắt đầu; client chơi giằng co rồi gửi dòng thời gian giữ/thả, máy chủ chạy lại đúng mô phỏng này để chốt.
// client/js/tug.js là bản sao 1-1 của các hàm dưới đây; TestTugMatchesClient (tug_parity_test.go) giữ hai bên khớp nhau.

// TugParams là độ khó giằng co của một bậc độ hiếm. Tiến độ và độ căng đều trong khoảng 0..1.
type TugParams struct {
	Tier        int     `json:"tier"`
	MaxSec      float64 `json:"maxSec"`      // quá giờ: giun thường vẫn bắt được, giun hiếm thoát
	Pull        float64 `json:"pull"`        // tiến độ/giây khi giữ lúc dây chùng
	Retreat     float64 `json:"retreat"`     // tiến độ mất/giây khi thả (giun rút lại)
	BaseStrain  float64 `json:"baseStrain"`  // độ căng tăng/giây khi giữ ngoài cơn giật
	BurstStrain float64 `json:"burstStrain"` // độ căng tăng thêm/giây × lực cơn giật
	Relax       float64 `json:"relax"`       // độ căng giảm/giây khi thả
	Gap         float64 `json:"gap"`         // khoảng nghỉ trung bình giữa hai cơn giật (giây)
	BurstMin    float64 `json:"burstMin"`    // lực cơn giật thấp nhất…
	BurstMax    float64 `json:"burstMax"`    // …và cao nhất
	Double      float64 `json:"double"`      // xác suất một cơn giật có cơn thứ hai liền sau
	CanEscape   bool    `json:"canEscape"`
	MaxSlips    int     `json:"maxSlips"`
}

// Bậc 0..4: Thường, Khá hiếm, Hiếm, Sử thi, Huyền thoại.
var TugTiers = []TugParams{
	{Tier: 0, MaxSec: 6, Pull: .55, Retreat: .04, BaseStrain: .12, BurstStrain: .5, Relax: 1.6, Gap: 1.8, BurstMin: .3, BurstMax: .5},
	{Tier: 1, MaxSec: 8, Pull: .42, Retreat: .05, BaseStrain: .16, BurstStrain: .8, Relax: 1.6, Gap: 1.5, BurstMin: .5, BurstMax: .8},
	{Tier: 2, MaxSec: 12, Pull: .3, Retreat: .06, BaseStrain: .2, BurstStrain: 1.25, Relax: 1.5, Gap: 1.3, BurstMin: .7, BurstMax: 1, CanEscape: true, MaxSlips: 3},
	{Tier: 3, MaxSec: 15, Pull: .3, Retreat: .06, BaseStrain: .21, BurstStrain: 1.3, Relax: 1.6, Gap: 1.35, BurstMin: .75, BurstMax: 1.1, Double: .25, CanEscape: true, MaxSlips: 3},
	{Tier: 4, MaxSec: 18, Pull: .25, Retreat: .07, BaseStrain: .2, BurstStrain: 1.25, Relax: 1.65, Gap: 1.4, BurstMin: .85, BurstMax: 1.15, Double: .28, CanEscape: true, MaxSlips: 3},
}

var rarityTiers = map[string]int{"Thường": 0, "Khá hiếm": 1, "Hiếm": 2, "Sử thi": 3, "Huyền thoại": 4}

// DigXP: đào cũng cho XP theo độ hiếm (ăn vẫn là nguồn XP chính).
var DigXP = []int{2, 4, 8, 15, 25}

const (
	TugStep      = 0.05 // giây mỗi bước mô phỏng
	TugSlipDrop  = 0.25 // tiến độ mất khi tuột
	TugSlipReset = 0.3  // độ căng sau khi tuột
	EscapeXP     = 5    // XP an ủi khi giun thoát
	ExhaustCap   = 0.85 // kiệt sức: 85% Thường, 15% Khá hiếm
)

func wormTier(w *Worm) int { return rarityTiers[w.Rarity] }

// Burst là một cơn giật của giun: bắt đầu At (giây từ lúc giằng co), dài Dur, lực Str.
type Burst struct {
	At  float64 `json:"at"`
	Dur float64 `json:"dur"`
	Str float64 `json:"str"`
}

// tugRand: PRNG 32-bit (mulberry32) để JavaScript tái tạo chính xác cùng dãy số.
type tugRand struct{ s uint32 }

func (r *tugRand) next() float64 {
	r.s += 0x6D2B79F5
	t := r.s
	t = (t ^ (t >> 15)) * (t | 1)
	t ^= t + (t^(t>>7))*(t|61)
	return float64(t^(t>>14)) / 4294967296
}

// TugBursts sinh lịch cơn giật từ seed; cùng seed cho cùng lịch ở máy chủ và client.
func TugBursts(seed uint32, p TugParams) []Burst {
	r := &tugRand{s: seed}
	var out []Burst
	t := 0.6 + r.next()*0.6
	for t < p.MaxSec {
		dur := 0.4 + r.next()*0.8
		str := p.BurstMin + r.next()*(p.BurstMax-p.BurstMin)
		out = append(out, Burst{At: round3(t), Dur: round3(dur), Str: round3(str)})
		t += dur
		if r.next() < p.Double {
			d2 := 0.3 + r.next()*0.4
			out = append(out, Burst{At: round3(t + 0.15), Dur: round3(d2), Str: round3(str * 0.9)})
			t += 0.15 + d2
		}
		t += p.Gap*0.5 + r.next()*p.Gap
	}
	return out
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// TugEvent: tại giây T (từ lúc giằng co) người chơi giữ (Hold) hoặc thả nút.
type TugEvent struct {
	T    float64
	Hold bool
}

type TugResult struct {
	Caught   bool    `json:"caught"`
	Escaped  bool    `json:"escaped"`
	Slips    int     `json:"slips"`
	End      float64 `json:"end"` // giây kết thúc theo mô phỏng
	Progress float64 `json:"progress"`
}

// burstForce là lực cơn giật đang diễn ra tại thời điểm t (0 nếu không có).
func burstForce(bursts []Burst, t float64) float64 {
	f := 0.0
	for _, b := range bursts {
		if t >= b.At && t < b.At+b.Dur && b.Str > f {
			f = b.Str
		}
	}
	return f
}

// SimulateTug chạy giằng co theo các bước cố định. Sau sự kiện cuối, trạng thái giữ/thả được giữ nguyên
// tới khi bắt được, giun thoát hoặc hết giờ, nên kết quả không phụ thuộc lúc client gửi tin.
func SimulateTug(p TugParams, bursts []Burst, events []TugEvent) TugResult {
	prog, tension, slips := 0.0, 0.0, 0
	hold, ei := false, 0
	steps := int(math.Ceil(p.MaxSec / TugStep))
	for i := 0; i < steps; i++ {
		t := float64(i) * TugStep
		for ei < len(events) && events[ei].T <= t+1e-9 {
			hold = events[ei].Hold
			ei++
		}
		f := burstForce(bursts, t)
		if hold {
			prog += p.Pull * TugStep * (1 - 0.6*tension)
			tension += (p.BaseStrain + p.BurstStrain*f) * TugStep
		} else {
			back := p.Retreat
			if f > 0 {
				back *= 1.6
			}
			prog -= back * TugStep
			tension -= p.Relax * TugStep
		}
		tension = math.Max(0, tension)
		prog = math.Max(0, math.Min(1, prog))
		if tension >= 1 {
			slips++
			prog = math.Max(0, prog-TugSlipDrop)
			tension = TugSlipReset
			if p.CanEscape && slips >= p.MaxSlips {
				return TugResult{Escaped: true, Slips: slips, End: round3(t + TugStep), Progress: prog}
			}
		}
		if prog >= 1 {
			return TugResult{Caught: true, Slips: slips, End: round3(t + TugStep), Progress: 1}
		}
	}
	// Hết giờ: giun thường/khá hiếm cuối cùng vẫn lôi lên được (không có ngõ cụt), giun hiếm thì thoát.
	return TugResult{Caught: !p.CanEscape, Escaped: p.CanEscape, Slips: slips, End: p.MaxSec, Progress: prog}
}

// ScaleTug đổi tốc độ toàn bộ giằng co (bộ cân bằng thử nghiệm chạy nhanh hơn để test e2e không chậm).
func ScaleTug(p TugParams, speed float64) TugParams {
	if speed == 1 || speed <= 0 {
		return p
	}
	p.MaxSec /= speed
	p.Pull *= speed
	p.Retreat *= speed
	p.BaseStrain *= speed
	p.BurstStrain *= speed
	p.Relax *= speed
	p.Gap /= speed
	return p
}

// PlanTug dựng một dòng thời gian của "người chơi hợp lý" phản xạ sau delay giây: thả khi độ căng vượt hi,
// giữ lại khi xuống dưới lo. Dùng cho test và cho bot kiểm thử tải.
func PlanTug(p TugParams, bursts []Burst, lo, hi, delay float64) []TugEvent {
	prog, tension := 0.0, 0.0
	hold := true
	events := []TugEvent{{0, true}}
	pending, pendAt := false, 0.0
	steps := int(math.Ceil(p.MaxSec / TugStep))
	for i := 0; i < steps; i++ {
		t := float64(i) * TugStep
		if pending && t+1e-9 >= pendAt {
			hold = !hold
			events = append(events, TugEvent{round3(t), hold})
			pending = false
		}
		f := burstForce(bursts, t)
		if hold {
			prog += p.Pull * TugStep * (1 - 0.6*tension)
			tension += (p.BaseStrain + p.BurstStrain*f) * TugStep
		} else {
			back := p.Retreat
			if f > 0 {
				back *= 1.6
			}
			prog -= back * TugStep
			tension -= p.Relax * TugStep
		}
		tension = math.Max(0, tension)
		prog = math.Max(0, math.Min(1, prog))
		if tension >= 1 {
			prog = math.Max(0, prog-TugSlipDrop)
			tension = TugSlipReset
		}
		if prog >= 1 {
			break
		}
		if !pending && ((hold && tension > hi) || (!hold && tension < lo)) {
			pending, pendAt = true, t+delay
		}
	}
	return events
}

// RollDigWorm gieo loài giun lúc bắt đầu đào. Kiệt sức: chỉ hai bậc đầu (85/15). luck > 0 (bạn đồng hành
// "Mỏ Thính") nhân tỉ lệ các bậc từ Khá hiếm trở lên; glow > 1 (ban đêm) nhân thêm tỉ lệ giun phát sáng.
// Giun thường co lại để tổng vẫn bằng 1.
func RollDigWorm(set string, r float64, now time.Time, exhausted bool, luck, glow float64) *Worm {
	tab := WormSets[set]
	if exhausted {
		if r < ExhaustCap {
			return &tab[0]
		}
		return &tab[1]
	}
	if luck <= 0 && glow <= 1 {
		return RollWormAt(set, r, now)
	}
	if s := CurrentSeason(now); s != nil && s.Worm != "" {
		if r < s.WormProb {
			return wormByID(s.Worm)
		}
		r = (r - s.WormProb) / (1 - s.WormProb)
	}
	probs := make([]float64, len(tab))
	rest := 0.0
	for i := 1; i < len(tab); i++ {
		probs[i] = tab[i].Prob * (1 + luck)
		if NightWorms[tab[i].ID] && glow > 1 {
			probs[i] *= glow
		}
		rest += probs[i]
	}
	probs[0] = 1 - rest
	acc := 0.0
	for i := range tab {
		acc += probs[i]
		if r < acc {
			return &tab[i]
		}
	}
	return &tab[len(tab)-1]
}
