package main

import (
	"fmt"
	"math"
)

// Mỗi trại là một bản đồ liền (PLAN_THE_GIOI_3D mục 6): các khu là vùng chữ nhật đặt ở tọa độ thật,
// nối với nhau bằng đường làng. Người chơi đi bộ giữa các khu, không có cổng dịch chuyển.
// Khu hiện tại của người chơi được tính từ vị trí; luật chơi (đào, chợ, PK…) vẫn kiểm tra theo khu.
// Server là nguồn duy nhất cho vị trí tương tác; client dựng cảnh trang trí quanh các điểm này.

type Vec struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
}

func (a Vec) Dist(b Vec) float64 { return math.Hypot(a.X-b.X, a.Z-b.Z) }

type Item struct {
	ID    string  `json:"id"`
	Kind  string  `json:"kind"`
	Label string  `json:"label"`
	Emo   string  `json:"emo"`
	X     float64 `json:"x"`
	Z     float64 `json:"z"`
	R     float64 `json:"r"`
	To    string  `json:"to,omitempty"`
	Y     float64 `json:"y,omitempty"`
}

func (it Item) Pos() Vec { return Vec{it.X, it.Z} }

// Zone là một khu. Items, Spawns, Obstacles và Entry đều ở tọa độ thế giới; HalfX/HalfZ là nửa kích thước
// quanh tâm (CX, CZ). Flip = khu xoay 180° so với bản vẽ gốc (lối vào quay về phía quảng trường).
type Zone struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	CX        float64    `json:"cx"`
	CZ        float64    `json:"cz"`
	Flip      bool       `json:"flip,omitempty"`
	HalfX     float64    `json:"halfX"`
	HalfZ     float64    `json:"halfZ"`
	Spawns    []Vec      `json:"spawns"`
	Items     []Item     `json:"items"`
	Capacity  int        `json:"capacity"`
	Obstacles []Obstacle `json:"obstacles"`
	Entry     *Vec       `json:"entry,omitempty"` // đầu đường dẫn vào khu, ngay ngoài mép khu
	Outdoor   bool       `json:"outdoor,omitempty"`
}

// Shared geometry for server movement, camera collision and client pathfinding.
type Obstacle struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
	D float64 `json:"d"`
	H float64 `json:"h"`
}

func (z *Zone) Contains(p Vec) bool {
	return math.Abs(p.X-z.CX) <= z.HalfX && math.Abs(p.Z-z.CZ) <= z.HalfZ
}

func (z *Zone) Walkable(p Vec) bool {
	if math.Abs(p.X-z.CX) > z.HalfX-.35 || math.Abs(p.Z-z.CZ) > z.HalfZ-.35 {
		return false
	}
	for _, o := range z.Obstacles {
		if math.Abs(p.X-o.X) < o.W/2+.35 && math.Abs(p.Z-o.Z) < o.D/2+.35 {
			return false
		}
	}
	return true
}

func (z *Zone) Move(from, to Vec) Vec {
	n := int(math.Ceil(from.Dist(to) / .2))
	if n < 1 {
		return from
	}
	if n > 100 {
		n = 100
	}
	dx, dz := (to.X-from.X)/float64(n), (to.Z-from.Z)/float64(n)
	for i := 0; i < n; i++ {
		q := Vec{from.X + dx, from.Z + dz}
		if z.Walkable(q) {
			from = q
			continue
		}
		q = Vec{from.X + dx, from.Z}
		if z.Walkable(q) {
			from = q
		}
		q = Vec{from.X, from.Z + dz}
		if z.Walkable(q) {
			from = q
		}
	}
	return from
}

func (z *Zone) Item(id string) *Item {
	for i := range z.Items {
		if z.Items[i].ID == id {
			return &z.Items[i]
		}
	}
	return nil
}

func (z *Zone) Clamp(p Vec) Vec {
	p.X = math.Max(z.CX-z.HalfX, math.Min(z.CX+z.HalfX, p.X))
	p.Z = math.Max(z.CZ-z.HalfZ, math.Min(z.CZ+z.HalfZ, p.Z))
	return p
}

// place đưa khu vẽ theo tọa độ cục bộ (tâm 0,0; lối vào ở phía +z) ra vị trí thật trên bản đồ.
func (z *Zone) place(cx, cz float64, flip bool) *Zone {
	tf := func(x, y float64) (float64, float64) {
		if flip {
			return cx - x, cz - y
		}
		return cx + x, cz + y
	}
	z.CX, z.CZ, z.Flip = cx, cz, flip
	for i := range z.Items {
		z.Items[i].X, z.Items[i].Z = tf(z.Items[i].X, z.Items[i].Z)
	}
	for i := range z.Spawns {
		z.Spawns[i].X, z.Spawns[i].Z = tf(z.Spawns[i].X, z.Spawns[i].Z)
	}
	for i := range z.Obstacles {
		z.Obstacles[i].X, z.Obstacles[i].Z = tf(z.Obstacles[i].X, z.Obstacles[i].Z)
	}
	ex, ez := 0.0, z.HalfZ+3
	if z.Entry != nil {
		ex, ez = z.Entry.X, z.Entry.Z
	}
	x, y := tf(ex, ez)
	z.Entry = &Vec{x, y}
	return z
}

const ZoneLobby = "lobby"

// Vùng (mục 8.5): trại Hypex gốc và trại Ven Suối. Mỗi vùng là một bản đồ liền có khu xuất phát riêng.
const (
	RegionHypex   = "hypex"
	RegionVenSuoi = "vensuoi"
)

type Region struct {
	ID      string
	Title   string
	Entry   string // khu xuất hiện khi tới vùng (bằng xe)
	Level   int    // cấp tối thiểu để vào vùng
	Themes  []string
	Zones   map[string]*Zone
	Station string // khu bến xe
	WormSet string // bảng giun của các ụ đất trong vùng
	Outdoor string // khu ngoài trời: mọi chỗ không thuộc khu nào khác
	Hub     Vec    // nút giao các con đường (quảng trường)
	World   *Zone  // biên bản đồ + toàn bộ vật cản, dùng cho di chuyển
}

// AreaAt trả về khu chứa điểm p; ngoài mọi khu là khu ngoài trời của vùng.
func (reg *Region) AreaAt(p Vec) string {
	for id, z := range reg.Zones {
		if !z.Outdoor && z.Contains(p) {
			return id
		}
	}
	return reg.Outdoor
}

// WorldInfo là phần dữ liệu bản đồ gửi cho client để dựng cảnh và tìm đường.
func (reg *Region) WorldInfo() map[string]any {
	w := reg.World
	return map[string]any{"cx": w.CX, "cz": w.CZ, "halfX": w.HalfX, "halfZ": w.HalfZ, "obstacles": w.Obstacles,
		"hub": reg.Hub, "outdoor": reg.Outdoor, "title": reg.Title}
}

func newRegion(r *Region) *Region {
	out := r.Zones[r.Outdoor]
	r.World = &Zone{ID: "world", CX: out.CX, CZ: out.CZ, HalfX: out.HalfX, HalfZ: out.HalfZ}
	for _, z := range r.Zones {
		if z.Capacity == 0 {
			z.Capacity = 25
		}
		r.World.Obstacles = append(r.World.Obstacles, z.Obstacles...)
	}
	return r
}

var Zones = buildHypex()

var Regions = map[string]*Region{
	RegionHypex: newRegion(&Region{ID: RegionHypex, Title: "Trại Hypex", Entry: ZoneLobby, Level: 1, Themes: RoomThemes, Zones: Zones,
		Station: "station", WormSet: RegionHypex, Outdoor: "village", Hub: Vec{0, 0}}),
	RegionVenSuoi: newRegion(&Region{ID: RegionVenSuoi, Title: "Trại Ven Suối", Entry: "vs_station", Level: 3, Themes: []string{"Trại Ven Suối"},
		Zones: buildVenSuoi(), Station: "vs_station", WormSet: RegionVenSuoi, Outdoor: "vs_field", Hub: Vec{0, 5.5}}),
}

// Trại Ven Suối (ảnh 11): bến xe ở phía nam, con đường đất dẫn lên đồng ven suối có bãi giun đặc sản.
func buildVenSuoi() map[string]*Zone {
	st := &Zone{ID: "vs_station", Title: "Bến Ven Suối", HalfX: 10.4, HalfZ: 8.0,
		Spawns: []Vec{{0, 5.6}, {-1.2, 5.6}, {1.2, 5.6}, {0, 4.4}},
		Items: []Item{
			{ID: "tram_bus", Kind: "bus_stop", Label: "Trạm xe buýt", Emo: "🚌", X: 2, Z: -4.5, R: 2.6},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 7, Z: 3, R: 2.4},
			{ID: "duong_lien_trai", Kind: "road", Label: "Đường liên trại", Emo: "🛣️", X: -9.0, Z: 3, R: 2.4},
		},
		Obstacles: []Obstacle{{-.9, -5.7, 2.2, 4.6, 2.2}, {4, -5.1, 4.2, 1.6, 2.6}},
		Entry:     &Vec{0, -11}}
	farm := &Zone{ID: "vensuoi", Title: "Đồng Ven Suối", HalfX: 11.4, HalfZ: 9.0,
		Spawns: []Vec{{0, 7}, {-1.2, 7}, {1.2, 7}, {0, 5.8}},
		Items: []Item{
			{ID: "ech_xanh", Kind: "npc_frog", Label: "Ếch Xanh", Emo: "🐸", X: 6, Z: 5, R: 2.4},
		},
		// Dòng suối chảy dọc bờ đông, chỉ qua được ở cầu gỗ (z = -5).
		Obstacles: []Obstacle{{9.4, -9, 2.6, 6, .1}, {9.4, 4, 2.6, 16, .1}}}
	for i, p := range [][2]float64{{-7, -3}, {-4, 0}, {-1, -2.5}, {3, 0.5}, {6, -2}, {-6, 3}, {2, 4}} {
		soil := "Bãi bồi"
		if p[1] < -1 {
			soil = "Bùn ven suối"
		}
		farm.Items = append(farm.Items, Item{ID: "dat_" + string(rune('a'+i)), Kind: "dig", Label: soil, Emo: "🪱", X: p[0], Z: p[1], R: 1.8})
	}
	field := &Zone{ID: "vs_field", Title: "Đường Ven Suối", HalfX: 60, HalfZ: 60, Outdoor: true, Spawns: []Vec{{0, 5}}, Items: []Item{}}
	return map[string]*Zone{st.place(0, 22, false).ID: st, farm.place(0, -12, false).ID: farm, field.ID: field}
}

// Khu cho phép lái xe: không lái trong sảnh, khu đào, nhà riêng và sân PK (mục 8.4).
var DriveZones = map[string]bool{"station": true, "garage": true, "market": true, "vs_station": true, "village": true,
	"neighborhood": true, "park": true, "vs_field": true}

// buildHypex dựng trại Hypex: sảnh chuồng ở phía bắc quảng trường, các khu tỏa ra theo đường làng.
// Thời gian đi bộ (4,2 đơn vị/giây): cửa sảnh → quảng trường ≈ 12 giây, quảng trường → vườn/chợ ≈ 18 giây,
// → bến xe ≈ 27 giây (PLAN_THE_GIOI_3D mục 6.4).
func buildHypex() map[string]*Zone {
	// Sảnh 32 × 42: bản 40 × 52 nhìn từ góc sinh hoạt bị trống, nên các hàng ổ được xếp chặt lại theo ảnh mẫu.
	lobby := &Zone{ID: ZoneLobby, Title: "Sảnh chuồng cộng đồng", HalfX: 16, HalfZ: 21, Capacity: 50,
		Items: []Item{
			{ID: "ga_cu_cuc", Kind: "npc_elder", Label: "Cụ Cục · Chủ chuồng", X: 0, Z: -17.3, Y: 1.2, R: 4.6},
			{ID: "ga_don_khach", Kind: "npc_guide", Label: "Gà Đón Khách", X: 5.6, Z: 16.2, R: 2.4},
			{ID: "bang_tin", Kind: "board", Label: "Bảng tin làng", X: -5.6, Z: 16.6, R: 2.4},
			{ID: "mang_an", Kind: "trough", Label: "Máng ăn chung", X: -13.4, Z: -2, R: 2.4},
		},
		Obstacles: []Obstacle{{0, -17, 11, 4, 1.2}, {-14.6, -2, 1.2, 8, .8}, {14.6, -2, 1.2, 8, .8},
			{-13.4, -18.6, 3.2, 3, 2.6}, {13.6, -18.8, 2.4, 2.4, 2.6}, {-14.4, 11.4, 1.6, 3.2, 1}, {14.4, 11.4, 1.6, 3.2, 1},
			// Tường gỗ bao quanh, cửa chính rộng 8 ở phía nam.
			{0, -21, 32.5, .5, 2.6}, {-16, 0, .5, 42.5, 2.6}, {16, 0, .5, 42.5, 2.6}, {-10.05, 21, 11.9, .5, 1.7}, {10.05, 21, 11.9, .5, 1.7}},
		Entry: &Vec{0, 24}}
	for row := 0; row < 3; row++ {
		for col := -3; col <= 3; col++ {
			lobby.Spawns = append(lobby.Spawns, Vec{float64(col) * 1.5, 11.8 + float64(row)*1.2})
		}
	}
	for row := 0; row < 5; row++ {
		for col, x := range []float64{-11.3, -8.6, -5.9, -3.2, 3.2, 5.9, 8.6, 11.3} {
			lobby.Items = append(lobby.Items, Item{ID: fmt.Sprintf("nest_%d_%d", row, col), Kind: "nest", Label: "Ngồi ổ rơm", X: x, Z: -10 + float64(row)*4.4, R: 1.3})
		}
	}

	spawns := func(z float64) []Vec { return []Vec{{0, z}, {-1.2, z}, {1.2, z}, {0, z - 1.2}} }

	garden := &Zone{ID: "garden", Title: "Vườn giun", HalfX: 10.4, HalfZ: 9.0, Spawns: spawns(6.6),
		Items: []Item{{ID: "lon_un", Kind: "npc_pig", Label: "Lợn Ủn", Emo: "🐷", X: -7, Z: 4, R: 2.4}},
		// Hàng rào ba phía, phía nam mở ra đường làng.
		Obstacles: []Obstacle{{-10.4, 0, .3, 18, 1}, {0, -9, 20.8, .3, 1}, {10.4, 0, .3, 18, 1}}}
	// Tiểu vùng đất vườn / đất ẩm / gốc cây: khác hình thức, cùng bảng cơ bản (mục 7.2.1).
	digSpots := []struct {
		x, z float64
		soil string
	}{{-5, -2, "Đất vườn"}, {-2, 1, "Đất vườn"}, {2, -1, "Đất vườn"}, {5.5, -3.5, "Đất ẩm"}, {7, 0, "Đất ẩm"}, {-0.5, -4.5, "Gốc cây"}, {4, 2.5, "Đất vườn"}}
	for i, s := range digSpots {
		garden.Items = append(garden.Items, Item{ID: "dat_" + string(rune('a'+i)), Kind: "dig", Label: s.soil, Emo: "🪱", X: s.x, Z: s.z, R: 1.8})
	}

	market := &Zone{ID: "market", Title: "Chợ thóc", HalfX: 11.4, HalfZ: 9.0, Spawns: spawns(6.6),
		Items: []Item{
			{ID: "sap_giun", Kind: "shop_sell", Label: "Sạp thu mua giun", Emo: "🐷", X: -7, Z: -4.2, R: 2.4},
			{ID: "quay_an", Kind: "shop_food", Label: "Quầy thức ăn", Emo: "🍚", X: 0, Z: -4.2, R: 2.4},
			{ID: "quay_do", Kind: "shop_fashion", Label: "Quầy thời trang", Emo: "👗", X: 7, Z: -4.2, R: 2.4},
		},
		Obstacles: []Obstacle{{-7, -5.8, 3.9, 1.5, 1}, {0, -5.8, 4.1, 1.5, 1}, {7, -5.8, 3.9, 1.5, 1}}}

	arena := &Zone{ID: "arena", Title: "Sân tỷ thí", HalfX: 9.4, HalfZ: 9.0, Spawns: spawns(6.6),
		Items:     []Item{{ID: "cho_muc", Kind: "pk", Label: "Chó Mực (trọng tài)", Emo: "🐶", X: -5, Z: -3, R: 2.6}},
		Obstacles: []Obstacle{{-8, 0, 2.6, 7, 1}, {8, 0, 2.6, 7, 1}}}

	// Trạm ấp trứng (ảnh 6, mục 12).
	hatchery := &Zone{ID: "hatchery", Title: "Trạm ấp trứng", HalfX: 9.4, HalfZ: 8.0, Spawns: spawns(6.2),
		Items:     []Item{{ID: "vit_bong", Kind: "npc_duck", Label: "Vịt Bông", Emo: "🦆", X: -4, Z: -2, R: 2.4}},
		Obstacles: []Obstacle{{0, -6.2, 12.6, 3.6, 3.4}}}

	// Bến xe Chim Chíp (ảnh 9, mục 8.5): đặt vé xe buýt, điểm đỗ, đường liên trại cho xe máy/ô tô.
	station := &Zone{ID: "station", Title: "Bến xe Chim Chíp", HalfX: 11.0, HalfZ: 8.0, Spawns: spawns(6.2),
		Items: []Item{
			{ID: "chim_chip", Kind: "npc_bird", Label: "Chim Chíp", Emo: "🐦", X: -6, Z: -3, R: 2.4},
			{ID: "tram_bus", Kind: "bus_stop", Label: "Trạm xe buýt", Emo: "🚌", X: 2, Z: -4.5, R: 2.6},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 7, Z: 3, R: 2.4},
			{ID: "duong_lien_trai", Kind: "road", Label: "Đường liên trại", Emo: "🛣️", X: -9.6, Z: 3, R: 2.4},
		},
		Obstacles: []Obstacle{{-.9, -5.7, 2.2, 4.6, 2.2}, {4, -5.1, 4.2, 1.6, 2.6}}}

	// Cửa hàng xe và điểm đỗ (ảnh 10, mục 8.4).
	garage := &Zone{ID: "garage", Title: "Cửa hàng xe", HalfX: 9.4, HalfZ: 8.0, Spawns: spawns(6.2),
		Items: []Item{
			{ID: "de_tho", Kind: "npc_goat", Label: "Dê Thợ Máy", Emo: "🐐", X: -5, Z: -3, R: 2.4},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 5, Z: -2, R: 2.4},
		},
		Obstacles: []Obstacle{{0, -7.6, 9.4, .8, 3}}}

	neighborhood := &Zone{ID: "neighborhood", Title: "Xóm Rơm Vàng", HalfX: 22, HalfZ: 22, Spawns: []Vec{{0, 18}, {1.5, 18}, {-1.5, 18}},
		Items: []Item{
			{ID: "bien_nha", Kind: "home_sign", Label: "Chuồng riêng & thăm bạn", X: -8, Z: 0, R: 3},
			{ID: "bo_moc", Kind: "npc_cow", Label: "Bò Mộc", X: -8, Z: 7, R: 2.6},
			{ID: "hoi_ga", Kind: "guild_board", Label: "Nhà Hội Gà", X: 9, Z: -8, R: 3},
		},
		Obstacles: []Obstacle{{-13, -12, 5, 7, 4}, {13, -12, 5, 7, 4}, {-13, 4, 5, 7, 4}, {13, 4, 5, 7, 4}, {12, -16, 7, 4, 4}}}

	park := &Zone{ID: "park", Title: "Vườn An Nhiên", HalfX: 26, HalfZ: 24, Spawns: []Vec{{0, 20}, {1.5, 20}, {-1.5, 20}},
		Items:     []Item{{ID: "hoi_cho", Kind: "board", Label: "Lễ hội của làng", X: 12, Z: 5, R: 2.6}},
		Obstacles: []Obstacle{{0, -2, 18, 12, .1}}}

	village := &Zone{ID: "village", Title: "Làng Hypex", HalfX: 115, HalfZ: 125, Outdoor: true, Capacity: 50,
		Spawns: []Vec{{0, 14}, {2, 14}, {-2, 14}}, Items: []Item{}, Obstacles: []Obstacle{{0, 0, 5, 5, 9}}}

	zones := []*Zone{
		lobby.place(0, -72, false), garden.place(-70, -30, false), arena.place(70, -30, false),
		market.place(-72, 40, true), park.place(80, 45, true), hatchery.place(10, 66, true),
		neighborhood.place(-28, 78, true), station.place(68, 100, true), garage.place(40, 108, true), village,
	}
	out := map[string]*Zone{}
	for _, z := range zones {
		out[z.ID] = z
	}
	return out
}

// Kind của item nằm trong zone hiện tại và người chơi đủ gần.
func nearKind(zone *Zone, pos Vec, kind string) *Item {
	for i := range zone.Items {
		it := &zone.Items[i]
		if it.Kind == kind && pos.Dist(it.Pos()) <= it.R+InteractSlack {
			return it
		}
	}
	return nil
}

func regionInfo() map[string]any {
	out := map[string]any{}
	for id, r := range Regions {
		out[id] = map[string]any{"title": r.Title, "level": r.Level, "entry": r.Entry}
	}
	return out
}

func zoneTitle(id string) string {
	for _, r := range Regions {
		if z := r.Zones[id]; z != nil {
			return z.Title
		}
	}
	return id
}
