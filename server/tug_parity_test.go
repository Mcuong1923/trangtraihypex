package main

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Client (client/js/tug.js) và máy chủ (dig.go) phải ra cùng lịch cơn giật và cùng kết quả giằng co
// (PLAN_LOGIC_GAME 4.2, 21): nếu lệch, người chơi thấy bắt được mà máy chủ báo giun thoát. Test chạy bản JS
// thật bằng Node; máy không có Node thì bỏ qua (đặt NODE=/đường/dẫn/node nếu node trong PATH hỏng).

type tugCase struct {
	Seed uint32       `json:"seed"`
	P    TugParams    `json:"p"`
	Ev   [][2]float64 `json:"ev"`
}

type jsTug struct {
	Bursts []Burst `json:"bursts"`
	Res    struct {
		Caught  bool    `json:"caught"`
		Escaped bool    `json:"escaped"`
		Slips   int     `json:"slips"`
		End     float64 `json:"end"`
	} `json:"res"`
}

func findNode(t *testing.T) string {
	for _, c := range []string{os.Getenv("NODE"), "node"} {
		if c == "" {
			continue
		}
		path, err := exec.LookPath(c)
		if err != nil {
			continue
		}
		if exec.Command(path, "--version").Run() == nil {
			return path
		}
	}
	t.Skip("không có Node chạy được; bỏ qua so khớp giằng co với client")
	return ""
}

func TestTugMatchesClient(t *testing.T) {
	node := findNode(t)
	src, err := filepath.Abs("../client/js/tug.js")
	if err != nil {
		t.Fatal(err)
	}
	var cases []tugCase
	toEv := func(ev []TugEvent) [][2]float64 {
		out := [][2]float64{}
		for _, e := range ev {
			h := 0.0
			if e.Hold {
				h = 1
			}
			out = append(out, [2]float64{e.T, h})
		}
		return out
	}
	rng := &tugRand{s: 7}
	for _, speed := range []float64{1, 10} {
		for _, base := range TugTiers {
			p := ScaleTug(base, speed)
			for _, seed := range []uint32{1, 42, 123456789, 2654435769, 4294967295} {
				bursts := TugBursts(seed, p)
				cases = append(cases,
					tugCase{seed, p, toEv(PlanTug(p, bursts, .4, .6, 0))},
					tugCase{seed, p, toEv(PlanTug(p, bursts, .2, .45, .25/speed))},
					tugCase{seed, p, [][2]float64{{0, 1}}},
					tugCase{seed, p, [][2]float64{}})
				// Bấm nhả ngẫu nhiên (nhưng tất định) như người chơi thật.
				var ev []TugEvent
				hold := false
				for at := 0.0; at < p.MaxSec; at += (.05 + rng.next()*.6) / speed {
					hold = !hold
					ev = append(ev, TugEvent{round3(at), hold})
				}
				cases = append(cases, tugCase{seed, p, toEv(ev)})
			}
		}
	}
	dir := t.TempDir()
	in, _ := json.Marshal(cases)
	if err := os.WriteFile(filepath.Join(dir, "cases.json"), in, 0o644); err != nil {
		t.Fatal(err)
	}
	harness := `import { readFileSync } from 'node:fs';
import { tugBursts, simulateTug } from ` + "'file://" + filepath.ToSlash(src) + "'" + `;
const cases = JSON.parse(readFileSync(process.argv[2], 'utf8'));
process.stdout.write(JSON.stringify(cases.map((c) => { const bursts = tugBursts(c.seed, c.p); return { bursts, res: simulateTug(c.p, bursts, c.ev) }; })));
`
	hp := filepath.Join(dir, "harness.mjs")
	if err := os.WriteFile(hp, []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(node, hp, filepath.Join(dir, "cases.json")).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("node: %v\n%s", err, ee.Stderr)
		}
		t.Fatal(err)
	}
	var got []jsTug
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("đọc kết quả node: %v", err)
	}
	if len(got) != len(cases) {
		t.Fatalf("node trả %d kết quả, cần %d", len(got), len(cases))
	}
	caught, escaped := 0, 0
	for i, c := range cases {
		bursts := TugBursts(c.Seed, c.P)
		if len(bursts) != len(got[i].Bursts) {
			t.Fatalf("ca %d (bậc %d, seed %d): Go %d cơn giật, JS %d", i, c.P.Tier, c.Seed, len(bursts), len(got[i].Bursts))
		}
		for j := range bursts {
			if bursts[j] != got[i].Bursts[j] {
				t.Fatalf("ca %d cơn %d: Go %+v, JS %+v", i, j, bursts[j], got[i].Bursts[j])
			}
		}
		var ev []TugEvent
		for _, e := range c.Ev {
			ev = append(ev, TugEvent{e[0], e[1] != 0})
		}
		want, js := SimulateTug(c.P, bursts, ev), got[i].Res
		if want.Caught != js.Caught || want.Escaped != js.Escaped || want.Slips != js.Slips || math.Abs(want.End-js.End) > 1e-9 {
			t.Fatalf("ca %d (bậc %d, seed %d): Go %+v, JS %+v", i, c.P.Tier, c.Seed, want, js)
		}
		if want.Caught {
			caught++
		}
		if want.Escaped {
			escaped++
		}
	}
	// Bộ ca phải phủ cả hai kết cục, không thì test không chứng minh được gì.
	if caught == 0 || escaped == 0 {
		t.Fatalf("bộ ca thiếu kết cục: %d bắt, %d thoát", caught, escaped)
	}
}
