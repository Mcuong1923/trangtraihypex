package main

import (
	"math"
	"sort"
)

// Đường làng phía máy chủ (PLAN_LOGIC_GAME 5.1, 7.1): vòng quảng trường, đường nhánh từ quảng trường ra cổng
// từng khu, và đường vành đai bán kính VanhDai quanh quảng trường. Xe buýt, xe ôm và xe NPC chạy trên đúng các
// đoạn này; client vẽ lại cùng dữ liệu (WorldInfo.roads) nên ai cũng thấy xe đi trên đường đất.

const (
	PlazaRing = 10.5 // bán kính vòng đường quanh quảng trường cây đa
	VanhDai   = 36.0 // bán kính đường vành đai (mọi khu đều cách quảng trường > 48 m)
)

type RoadStop struct {
	ID    string  `json:"id"`    // mã khu có bến ở cổng
	Title string  `json:"title"` // tên khu
	Pos   Vec     `json:"pos"`   // chỗ đứng chờ xe, sát cổng khu
	S     float64 `json:"s"`     // vị trí trên tuyến xe buýt (mét từ đầu tuyến)
}

type RoadNet struct {
	Nodes []Vec      `json:"nodes"`
	Edges [][2]int   `json:"edges"`
	Stops []RoadStop `json:"stops"`
	Bus   []Vec      `json:"bus"` // tuyến xe buýt vòng làng (khép kín)
	BusS  []float64  `json:"-"`   // độ dài cộng dồn tới từng điểm của tuyến
	adj   map[int][]roadAdj
	entry map[string]int // mã khu → nút cổng khu
}

type roadAdj struct {
	to int
	d  float64
}

func (n *RoadNet) node(v Vec) int {
	for i, p := range n.Nodes {
		if p.Dist(v) < 0.05 {
			return i
		}
	}
	n.Nodes = append(n.Nodes, v)
	return len(n.Nodes) - 1
}

func (n *RoadNet) link(a, b int) {
	if a == b {
		return
	}
	n.Edges = append(n.Edges, [2]int{a, b})
	d := n.Nodes[a].Dist(n.Nodes[b])
	n.adj[a] = append(n.adj[a], roadAdj{b, d})
	n.adj[b] = append(n.adj[b], roadAdj{a, d})
}

func polar(hub Vec, r, ang float64) Vec { return Vec{hub.X + math.Cos(ang)*r, hub.Z + math.Sin(ang)*r} }

// buildRoads dựng đồ thị đường cho một vùng có quảng trường (Hub) và các khu có cổng.
func buildRoads(reg *Region) *RoadNet {
	n := &RoadNet{adj: map[int][]roadAdj{}, entry: map[string]int{}}
	hub := reg.Hub
	type spoke struct {
		id, title string
		ang       float64
		entry     Vec
	}
	var spokes []spoke
	for id, z := range reg.Zones {
		if z.Outdoor || z.Entry == nil {
			continue
		}
		spokes = append(spokes, spoke{id, z.Title, math.Atan2(z.Entry.Z-hub.Z, z.Entry.X-hub.X), *z.Entry})
	}
	sort.Slice(spokes, func(i, j int) bool { return spokes[i].ang < spokes[j].ang })
	if len(spokes) == 0 {
		return n
	}
	far := 0.0
	for _, s := range spokes {
		far = math.Max(far, s.entry.Dist(hub))
	}
	// Vùng nhỏ (Ven Suối) không có vành đai: chỉ quảng trường và đường nhánh.
	ring := far > VanhDai+8
	plaza := make([]int, len(spokes))
	belt := make([]int, len(spokes))
	for i, s := range spokes {
		plaza[i] = n.node(polar(hub, PlazaRing, s.ang))
		e := n.node(s.entry)
		n.entry[s.id] = e
		if ring && s.entry.Dist(hub) > VanhDai+4 {
			belt[i] = n.node(polar(hub, VanhDai, s.ang))
			n.link(plaza[i], belt[i])
			n.link(belt[i], e)
		} else {
			belt[i] = -1
			n.link(plaza[i], e)
		}
	}
	// Vòng quảng trường và vành đai: cung tròn chia nhỏ ~6 m một đoạn để xe đi tròn trịa.
	arc := func(r float64, a0, a1 float64, from, to int) {
		if a1 < a0 {
			a1 += 2 * math.Pi
		}
		steps := max(1, int(math.Ceil(r*(a1-a0)/6)))
		prev := from
		for k := 1; k < steps; k++ {
			mid := n.node(polar(hub, r, a0+(a1-a0)*float64(k)/float64(steps)))
			n.link(prev, mid)
			prev = mid
		}
		n.link(prev, to)
	}
	for i := range spokes {
		j := (i + 1) % len(spokes)
		if len(spokes) > 1 {
			arc(PlazaRing, spokes[i].ang, spokes[j].ang, plaza[i], plaza[j])
		}
	}
	if ring {
		var idx []int
		for i := range spokes {
			if belt[i] >= 0 {
				idx = append(idx, i)
			}
		}
		for k, i := range idx {
			j := idx[(k+1)%len(idx)]
			arc(VanhDai, spokes[i].ang, spokes[j].ang, belt[i], belt[j])
		}
	}
	// Tuyến xe buýt: theo vành đai (chiều kim đồng hồ trên bản đồ = góc tăng), tới mỗi nhánh thì rẽ ra
	// bến ở cổng khu rồi quay lại vành đai. Vùng không có vành đai thì vòng qua quảng trường.
	for i, s := range spokes {
		j := (i + 1) % len(spokes)
		junction := plaza[i]
		r := PlazaRing
		if ring && belt[i] >= 0 {
			junction, r = belt[i], VanhDai
		}
		stopPos := n.Nodes[n.entry[s.id]]
		// Bến đứng lệch ra mép đường một chút, cách cổng 2,5 m về phía quảng trường.
		dir := Vec{hub.X - stopPos.X, hub.Z - stopPos.Z}
		l := math.Hypot(dir.X, dir.Z)
		stand := Vec{stopPos.X + dir.X/l*2.5, stopPos.Z + dir.Z/l*2.5}
		n.Bus = append(n.Bus, n.Nodes[junction])
		n.Stops = append(n.Stops, RoadStop{ID: s.id, Title: s.title, Pos: stand, S: -1})
		n.Bus = append(n.Bus, stand)
		n.Bus = append(n.Bus, n.Nodes[junction])
		// Cung tới nhánh kế.
		a0, a1 := s.ang, spokes[j].ang
		if a1 <= a0 {
			a1 += 2 * math.Pi
		}
		if len(spokes) == 1 {
			a1 = a0 + 2*math.Pi
		}
		steps := max(1, int(math.Ceil(r*(a1-a0)/6)))
		for k := 1; k < steps; k++ {
			n.Bus = append(n.Bus, polar(hub, r, a0+(a1-a0)*float64(k)/float64(steps)))
		}
	}
	n.BusS = make([]float64, len(n.Bus)+1)
	si := 0
	for i := range n.Bus {
		next := n.Bus[(i+1)%len(n.Bus)]
		n.BusS[i+1] = n.BusS[i] + n.Bus[i].Dist(next)
		if si < len(n.Stops) && n.Stops[si].S < 0 && n.Bus[i] == n.Stops[si].Pos {
			n.Stops[si].S = n.BusS[i]
			si++
		}
	}
	return n
}

// BusLen là chiều dài một vòng tuyến xe buýt.
func (n *RoadNet) BusLen() float64 {
	if len(n.BusS) == 0 {
		return 0
	}
	return n.BusS[len(n.BusS)-1]
}

// BusAt trả điểm và hướng trên tuyến tại vị trí s (mét, quay vòng).
func (n *RoadNet) BusAt(s float64) (Vec, float64) {
	L := n.BusLen()
	if L == 0 {
		return Vec{}, 0
	}
	s = math.Mod(math.Mod(s, L)+L, L)
	i := sort.SearchFloat64s(n.BusS, s)
	if i > 0 {
		i--
	}
	if i >= len(n.Bus) {
		i = len(n.Bus) - 1
	}
	a, b := n.Bus[i], n.Bus[(i+1)%len(n.Bus)]
	seg := n.BusS[i+1] - n.BusS[i]
	k := 0.0
	if seg > 0 {
		k = (s - n.BusS[i]) / seg
	}
	return Vec{a.X + (b.X-a.X)*k, a.Z + (b.Z-a.Z)*k}, math.Atan2(b.X-a.X, b.Z-a.Z)
}

// nearest trả nút đường gần điểm p nhất.
func (n *RoadNet) nearest(p Vec) int {
	best, bd := 0, math.MaxFloat64
	for i, v := range n.Nodes {
		if d := v.Dist(p); d < bd {
			best, bd = i, d
		}
	}
	return best
}

// Route tìm đường ngắn nhất (Dijkstra) từ điểm from tới cổng khu dest; trả về đường gấp khúc và độ dài.
func (n *RoadNet) Route(from Vec, dest string) ([]Vec, float64, bool) {
	goal, ok := n.entry[dest]
	if !ok || len(n.Nodes) == 0 {
		return nil, 0, false
	}
	start := n.nearest(from)
	dist := make([]float64, len(n.Nodes))
	prev := make([]int, len(n.Nodes))
	done := make([]bool, len(n.Nodes))
	for i := range dist {
		dist[i], prev[i] = math.MaxFloat64, -1
	}
	dist[start] = 0
	for {
		u, best := -1, math.MaxFloat64
		for i, d := range dist {
			if !done[i] && d < best {
				u, best = i, d
			}
		}
		if u < 0 || u == goal {
			break
		}
		done[u] = true
		for _, e := range n.adj[u] {
			if nd := dist[u] + e.d; nd < dist[e.to] {
				dist[e.to], prev[e.to] = nd, u
			}
		}
	}
	if dist[goal] == math.MaxFloat64 {
		return nil, 0, false
	}
	var path []Vec
	for v := goal; v >= 0; v = prev[v] {
		path = append([]Vec{n.Nodes[v]}, path...)
	}
	path = append([]Vec{from}, path...)
	total := 0.0
	for i := 1; i < len(path); i++ {
		total += path[i-1].Dist(path[i])
	}
	return path, total, true
}

// RandomEdgePath: một đoạn đường ngẫu nhiên vài nút để xe NPC chạy rong, bắt đầu từ nút gần p.
func (n *RoadNet) RandomEdgePath(p Vec, hops int, rnd func() float64) []Vec {
	if len(n.Nodes) == 0 {
		return nil
	}
	cur, prev := n.nearest(p), -1
	path := []Vec{n.Nodes[cur]}
	for h := 0; h < hops; h++ {
		opts := n.adj[cur]
		if len(opts) == 0 {
			break
		}
		next := opts[int(rnd()*float64(len(opts)))%len(opts)].to
		if next == prev && len(opts) > 1 {
			next = opts[(int(rnd()*float64(len(opts)))+1)%len(opts)].to
		}
		prev, cur = cur, next
		path = append(path, n.Nodes[cur])
	}
	return path
}
