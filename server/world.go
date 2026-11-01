package main

import "math"

// Bố cục một chuồng công cộng: sảnh + sân liền kề dùng chung một phòng (mục 6.1, 6.4).
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
}

func (it Item) Pos() Vec { return Vec{it.X, it.Z} }

type Zone struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	HalfX  float64 `json:"halfX"`
	HalfZ  float64 `json:"halfZ"`
	Spawns []Vec   `json:"spawns"`
	Items  []Item  `json:"items"`
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
	p.X = math.Max(-z.HalfX, math.Min(z.HalfX, p.X))
	p.Z = math.Max(-z.HalfZ, math.Min(z.HalfZ, p.Z))
	return p
}

const ZoneLobby = "lobby"

var Zones = buildZones()

// Vùng (mục 8.5): trại Hypex gốc và trại Ven Suối. Mỗi vùng có bộ khu và khu xuất phát riêng.
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
}

var Regions = map[string]*Region{
	RegionHypex:   {ID: RegionHypex, Title: "Trại Hypex", Entry: ZoneLobby, Level: 1, Themes: RoomThemes, Zones: Zones, Station: "station", WormSet: RegionHypex},
	RegionVenSuoi: {ID: RegionVenSuoi, Title: "Trại Ven Suối", Entry: "vs_station", Level: 3, Themes: []string{"Trại Ven Suối"}, Zones: buildVenSuoi(), Station: "vs_station", WormSet: RegionVenSuoi},
}

// Trại Ven Suối (ảnh 11): bến xe ở đầu làng và đồng ven suối có bãi giun đặc sản.
func buildVenSuoi() map[string]*Zone {
	st := &Zone{ID: "vs_station", Title: "Bến Ven Suối", HalfX: 10.4, HalfZ: 8.0,
		Spawns: []Vec{{0, 5.6}, {-1.2, 5.6}, {1.2, 5.6}, {0, 4.4}},
		Items: []Item{
			{ID: "tram_bus", Kind: "bus_stop", Label: "Trạm xe buýt", Emo: "🚌", X: 2, Z: -4.5, R: 2.6},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 7, Z: 3, R: 2.4},
			{ID: "duong_lien_trai", Kind: "road", Label: "Đường liên trại", Emo: "🛣️", X: -9.0, Z: 3, R: 2.4},
			{ID: "ra_dong", Kind: "gate", Label: "Ra đồng Ven Suối", Emo: "🌾", X: -4, Z: -7.4, R: 2.2, To: "vensuoi"},
		}}
	farm := &Zone{ID: "vensuoi", Title: "Đồng Ven Suối", HalfX: 11.4, HalfZ: 9.0,
		Spawns: []Vec{{0, 7}, {-1.2, 7}, {1.2, 7}, {0, 5.8}},
		Items: []Item{
			{ID: "ech_xanh", Kind: "npc_frog", Label: "Ếch Xanh", Emo: "🐸", X: 6, Z: 5, R: 2.4},
			{ID: "ve_ben", Kind: "gate", Label: "Về Bến Ven Suối", Emo: "🚌", X: 0, Z: 8.4, R: 2.2, To: "vs_station"},
		}}
	for i, p := range [][2]float64{{-7, -3}, {-4, 0}, {-1, -2.5}, {3, 0.5}, {6, -2}, {-6, 3}, {2, 4}} {
		soil := "Bãi bồi"
		if p[1] < -1 {
			soil = "Bùn ven suối"
		}
		farm.Items = append(farm.Items, Item{ID: "dat_" + string(rune('a'+i)), Kind: "dig", Label: soil, Emo: "🪱", X: p[0], Z: p[1], R: 1.8})
	}
	return map[string]*Zone{st.ID: st, farm.ID: farm}
}

// Khu cho phép lái xe: không lái trong sảnh, khu đào, nhà riêng và sân PK (mục 8.4).
var DriveZones = map[string]bool{"station": true, "garage": true, "market": true, "vs_station": true}

func buildZones() map[string]*Zone {
	gate := func(id, label, emo string, x, z float64, to string) Item {
		return Item{ID: id, Kind: "gate", Label: label, Emo: emo, X: x, Z: z, R: 2.2, To: to}
	}
	back := gate("ve_sanh", "Về sảnh chuồng", "🏠", 0, 8.4, ZoneLobby)

	// Nhiều điểm xuất hiện rải trên thảm rơm cửa vào để gà mới không chồng lên nhau (mục 6.4).
	var lobbySpawns []Vec
	for i := -3; i <= 3; i++ {
		lobbySpawns = append(lobbySpawns, Vec{float64(i) * 1.3, 8.2}, Vec{float64(i)*1.3 + 0.65, 6.9})
	}

	lobby := &Zone{ID: ZoneLobby, Title: "Sảnh chuồng gà", HalfX: 12.4, HalfZ: 9.6, Spawns: lobbySpawns,
		Items: []Item{
			{ID: "ga_cu_cuc", Kind: "npc_elder", Label: "Gà Cụ Cục", Emo: "🐔", X: -2.5, Z: 5, R: 2.4},
			{ID: "bang_tin", Kind: "board", Label: "Bảng tin", Emo: "📋", X: 2.5, Z: 5, R: 2.2},
			{ID: "mang_an", Kind: "trough", Label: "Máng ăn", Emo: "🌾", X: -10.6, Z: 2.5, R: 2.2},
			// Bảng chỉ đường về chuồng riêng: về nhà, thăm bạn, xem chuồng mở cửa (mục 6.4).
			{ID: "bien_nha", Kind: "home_sign", Label: "Chuồng riêng & thăm bạn", Emo: "🏡", X: 8.6, Z: 5.6, R: 2.2},
			gate("ra_vuon", "Ra Vườn giun", "🪱", -8.5, -8.6, "garden"),
			gate("ra_cho", "Ra Chợ thóc", "🌾", 0, -8.6, "market"),
			gate("ra_san", "Ra Sân tỷ thí", "🥊", 8.5, -8.6, "arena"),
			// V1.1: cửa hai bên vách ra trạm ấp trứng và bến xe (mục 6.1).
			gate("ra_ap", "Ra Trạm ấp trứng", "🥚", -11.6, 6.6, "hatchery"),
			gate("ra_ben", "Ra Bến xe Chim Chíp", "🚌", 11.6, -2.6, "station"),
		}}

	garden := &Zone{ID: "garden", Title: "Vườn giun", HalfX: 10.4, HalfZ: 9.0,
		Spawns: []Vec{{0, 6.6}, {-1.2, 6.6}, {1.2, 6.6}, {0, 5.4}},
		Items: []Item{
			{ID: "lon_un", Kind: "npc_pig", Label: "Lợn Ủn", Emo: "🐷", X: -7, Z: 4, R: 2.4},
			back,
		}}
	// Tiểu vùng đất vườn / đất ẩm / gốc cây: khác hình thức, cùng bảng cơ bản (mục 7.2.1).
	digSpots := []struct {
		x, z float64
		soil string
	}{{-5, -2, "Đất vườn"}, {-2, 1, "Đất vườn"}, {2, -1, "Đất vườn"}, {5.5, -3.5, "Đất ẩm"}, {7, 0, "Đất ẩm"}, {-0.5, -4.5, "Gốc cây"}, {4, 2.5, "Đất vườn"}}
	for i, s := range digSpots {
		garden.Items = append(garden.Items, Item{ID: "dat_" + string(rune('a'+i)), Kind: "dig", Label: s.soil, Emo: "🪱", X: s.x, Z: s.z, R: 1.8})
	}

	market := &Zone{ID: "market", Title: "Chợ thóc", HalfX: 11.4, HalfZ: 9.0,
		Spawns: []Vec{{0, 6.6}, {-1.2, 6.6}, {1.2, 6.6}, {0, 5.4}},
		Items: []Item{
			{ID: "sap_giun", Kind: "shop_sell", Label: "Sạp thu mua giun", Emo: "🐷", X: -7, Z: -4.2, R: 2.4},
			{ID: "quay_an", Kind: "shop_food", Label: "Quầy thức ăn", Emo: "🍚", X: 0, Z: -4.2, R: 2.4},
			{ID: "quay_do", Kind: "shop_fashion", Label: "Quầy thời trang", Emo: "👗", X: 7, Z: -4.2, R: 2.4},
			back,
		}}

	arena := &Zone{ID: "arena", Title: "Sân tỷ thí", HalfX: 9.4, HalfZ: 9.0,
		Spawns: []Vec{{0, 6.6}, {-1.2, 6.6}, {1.2, 6.6}, {0, 5.4}},
		Items: []Item{
			{ID: "cho_muc", Kind: "pk", Label: "Chó Mực (trọng tài)", Emo: "🐶", X: -5, Z: -3, R: 2.6},
			back,
		}}

	// Trạm ấp trứng (ảnh 6, mục 12).
	hatchery := &Zone{ID: "hatchery", Title: "Trạm ấp trứng", HalfX: 9.4, HalfZ: 8.0,
		Spawns: []Vec{{0, 6.2}, {-1.2, 6.2}, {1.2, 6.2}, {0, 5.0}},
		Items: []Item{
			{ID: "vit_bong", Kind: "npc_duck", Label: "Vịt Bông", Emo: "🦆", X: -4, Z: -2, R: 2.4},
			{ID: "ve_sanh", Kind: "gate", Label: "Về sảnh chuồng", Emo: "🏠", X: 0, Z: 7.4, R: 2.2, To: ZoneLobby},
		}}
	// Bến xe Chim Chíp (ảnh 9, mục 8.5): đặt vé xe buýt, điểm đỗ, đường liên trại cho xe máy/ô tô.
	station := &Zone{ID: "station", Title: "Bến xe Chim Chíp", HalfX: 11.0, HalfZ: 8.0,
		Spawns: []Vec{{0, 6.2}, {-1.2, 6.2}, {1.2, 6.2}, {0, 5.0}},
		Items: []Item{
			{ID: "chim_chip", Kind: "npc_bird", Label: "Chim Chíp", Emo: "🐦", X: -6, Z: -3, R: 2.4},
			{ID: "tram_bus", Kind: "bus_stop", Label: "Trạm xe buýt", Emo: "🚌", X: 2, Z: -4.5, R: 2.6},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 7, Z: 3, R: 2.4},
			{ID: "duong_lien_trai", Kind: "road", Label: "Đường liên trại", Emo: "🛣️", X: -9.6, Z: 3, R: 2.4},
			{ID: "ra_xe", Kind: "gate", Label: "Ra Cửa hàng xe", Emo: "🛵", X: 10.2, Z: -6.6, R: 2.2, To: "garage"},
			{ID: "ve_sanh", Kind: "gate", Label: "Về sảnh chuồng", Emo: "🏠", X: 0, Z: 7.4, R: 2.2, To: ZoneLobby},
		}}
	// Cửa hàng xe và điểm đỗ (ảnh 10, mục 8.4).
	garage := &Zone{ID: "garage", Title: "Cửa hàng xe", HalfX: 9.4, HalfZ: 8.0,
		Spawns: []Vec{{0, 6.2}, {-1.2, 6.2}, {1.2, 6.2}, {0, 5.0}},
		Items: []Item{
			{ID: "de_tho", Kind: "npc_goat", Label: "Dê Thợ Máy", Emo: "🐐", X: -5, Z: -3, R: 2.4},
			{ID: "diem_do", Kind: "parking", Label: "Điểm đỗ xe", Emo: "🅿️", X: 5, Z: -2, R: 2.4},
			{ID: "ve_ben", Kind: "gate", Label: "Về Bến xe", Emo: "🚌", X: 0, Z: 7.4, R: 2.2, To: "station"},
		}}

	return map[string]*Zone{lobby.ID: lobby, garden.ID: garden, market.ID: market, arena.ID: arena,
		hatchery.ID: hatchery, station.ID: station, garage.ID: garage}
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
