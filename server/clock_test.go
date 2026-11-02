package main

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"
)

// Ngày/đêm và thời tiết (PLAN_LOGIC_GAME 16): ngày game 48 phút chia đôi, mưa khoảng 10% thời gian, đêm
// Giun phát sáng ×1,5 mà bảng giun vẫn cộng đủ 1, mưa thì hồi đất −30% và xe ôm phụ thu 1 thóc.

func TestGameClockDayNightAndRain(t *testing.T) {
	start := time.Unix(1_800_000_000-1_800_000_000%int64(GameDay.Seconds()), 0)
	if c := GameClock(start); c.Phase != 0 || c.Night || c.NextIn != GameDay.Seconds()/2 {
		t.Fatalf("đầu ngày game: %+v", c)
	}
	if c := GameClock(start.Add(GameDay / 2)); !c.Night || math.Abs(c.Phase-0.5) > 1e-9 {
		t.Fatalf("nửa sau là đêm: %+v", c)
	}
	if daypart(start.Add(GameDay/4)) != "buổi trưa" || daypart(start.Add(GameDay*3/4)) != "buổi tối" {
		t.Fatal("lời chào theo buổi")
	}
	// Mưa: cùng một khung thì mọi phòng thấy như nhau; cả năm khoảng 10% thời gian.
	rain, n := 0, 0
	for at := start; at.Before(start.AddDate(1, 0, 0)); at = at.Add(RainSlot) {
		c := GameClock(at)
		if c.Rain != GameClock(at.Add(RainSlot-time.Second)).Rain {
			t.Fatal("thời tiết đổi giữa khung")
		}
		if c.RainLeft <= 0 || c.RainLeft > 50*RainSlot.Seconds() {
			t.Fatalf("thời gian tới lúc đổi thời tiết: %v", c.RainLeft)
		}
		if c.Rain {
			rain++
		}
		n++
	}
	if f := float64(rain) / float64(n); f < 0.08 || f > 0.12 {
		t.Fatalf("mưa %.1f%% thời gian, cần khoảng 10%%", f*100)
	}
}

func TestNightGlowKeepsTableNormalised(t *testing.T) {
	r := rand.New(rand.NewPCG(5, 6))
	day := t0.AddDate(1, 0, 0) // ngoài mùa sự kiện
	const N = 200000
	glow := 0
	for i := 0; i < N; i++ {
		if w := RollDigWorm(RegionHypex, r.Float64(), day, false, 0, NightGlow); w.ID == "sang" {
			glow++
		}
	}
	want := Worms[2].Prob * NightGlow
	if got := float64(glow) / N; math.Abs(got-want) > 0.003 {
		t.Fatalf("đêm: Giun phát sáng %.4f, cần %.4f", got, want)
	}
	// Bảng sau khi nhân vẫn cộng đủ 1 (giun thường co lại) và kiệt sức thì đêm cũng không ra giun hiếm.
	if w := RollDigWorm(RegionHypex, 0.999999, day, false, LuckCap, NightGlow); w == nil {
		t.Fatal("cuối bảng phải có giun")
	}
	if w := RollDigWorm(RegionHypex, 0.99, day, true, 0, NightGlow); wormTier(w) > 1 {
		t.Fatal("kiệt sức chỉ ra giun bậc thấp, kể cả ban đêm")
	}
}

func TestRainXeOmSurcharge(t *testing.T) {
	a := newAcc()
	var dry, wet time.Time
	for at := t0; dry.IsZero() || wet.IsZero(); at = at.Add(RainSlot) {
		if GameClock(at).Rain {
			wet = at
		} else {
			dry = at
		}
	}
	if xeomFare(a, 50, dry) != 1 || xeomFare(a, 200, dry) != 2 {
		t.Fatal("trời khô: 1–2 thóc theo quãng đường")
	}
	if xeomFare(a, 50, wet) != 1+RainXeOmFare || xeomFare(a, 200, wet) != 2+RainXeOmFare {
		t.Fatal("trời mưa: phụ thu khăn lau")
	}
}
