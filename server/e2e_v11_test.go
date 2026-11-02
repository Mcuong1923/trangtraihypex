package main

import (
	"strings"
	"testing"
	"time"
)

// End-to-end V1.1/V1.2 qua WebSocket: xe buýt sang Ven Suối, xe riêng chở bạn liên trại,
// giao dịch, hợp đồng thợ xây, Hội Gà.

// seeded tạo người chơi với tài khoản được chỉnh trước khi vào game (chưa phòng nào giữ tài khoản).
func (e *env) seeded(t *testing.T, user, name string, f func(*Account)) (*client, string) {
	tok, _ := e.registerFull(t, user)
	id, _ := e.store.AccountIDForToken(tok)
	acc, _ := e.store.Account(id)
	acc.Char = &Character{Name: name, Color: "vang", Comb: "don", Shirt: "none", Sex: "an", Tail: "quat", Eyes: "tron"}
	f(acc)
	e.store.Save(acc)
	e.store.Social.Ensure(id, name)
	c := e.dial(t, user, tok)
	c.expect("welcome", 2*time.Second)
	return c, id
}

func (c *client) region() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.zone
}

func TestBusToVenSuoi(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "anan", "Gà An", func(a *Account) { a.Level = 3 })
	newbie, _ := e.seeded(t, "bebe", "Gà Bé", func(a *Account) {})

	a.goZone("ra_ben", "station")
	a.walkToItem("station", "tram_bus")
	r := a.mustOK("book", nil)
	if !strings.Contains(r["msg"].(string), "miễn phí") {
		t.Fatalf("lượt đầu dùng miễn phí: %v", r["msg"])
	}
	e.warp(newbie, "station")
	newbie.walkToItem("station", "tram_bus")
	if r := newbie.act("book", nil); r["ok"] == true {
		t.Fatal("cấp 1 chưa tới được Ven Suối")
	}
	a.expect("trip", 3*time.Second)
	w := a.expect("welcome", 3*time.Second)
	if w["zone"] != "vs_station" || !strings.Contains(w["room"].(map[string]any)["name"].(string), "Ven Suối") {
		t.Fatalf("tới Bến Ven Suối: %v %v", w["zone"], w["room"])
	}
	me := w["me"].(map[string]any)
	if me["booking"] != nil || me["tripsLeft"].(float64) != MaxBusTokens-1 {
		t.Fatalf("quyết toán chuyến: %v còn %v lượt", me["booking"], me["tripsLeft"])
	}

	// Đào ở đồng Ven Suối ra giun đặc sản.
	a.goZone("ra_dong", "vensuoi")
	a.walkToItem("vensuoi", "dat_b")
	loot := a.dig("dat_b")["data"].(map[string]any)["loot"].(map[string]any)
	if reg := wormByID(loot["worm"].(string)).Region; reg != RegionVenSuoi && reg != "event" {
		t.Fatalf("giun ở Ven Suối phải thuộc bảng đặc sản: %v", loot["worm"])
	}

	// "Về sảnh chuồng Hypex" miễn phí.
	a.mustOK("go", msg{"to": "lobby", "id": "home"})
	if w := a.expect("welcome", 3*time.Second); w["zone"] != ZoneLobby || strings.Contains(w["room"].(map[string]any)["name"].(string), "Ven Suối") {
		t.Fatal("về sảnh Hypex")
	}
}

func TestCarCarriesFriendAcrossRegions(t *testing.T) {
	e := newEnv(t, t.TempDir())
	d, _ := e.seeded(t, "lai", "Gà Lái", func(a *Account) { a.Level = 7; a.Thoc = 20000 })
	p, _ := e.seeded(t, "khach", "Gà Khách", func(a *Account) { a.Level = 3 })
	e.warp(d, "garage")
	d.walkToItem("garage", "de_tho")
	d.mustOK("vehicle_buy", msg{"id": "o_to"})
	d.mustOK("vehicle_buy", msg{"id": "xe_dap"})
	d.walkToItem("garage", "diem_do")
	d.mustOK("vehicle_out", msg{"id": "o_to"})
	if r := d.act("vehicle_out", msg{"id": "xe_dap"}); r["ok"] == true {
		t.Fatal("mỗi người tối đa một xe đang ra")
	}

	e.warp(p, "garage")
	pos := d.position()
	p.walkTo(pos.X+1, pos.Z+0.5)
	var pPID string
	for {
		m := d.expect("pjoin", 2*time.Second)
		if pp := m["p"].(map[string]any); pp["name"] == "Gà Khách" {
			pPID = pp["pid"].(string)
			break
		}
	}
	d.mustOK("seat_invite", msg{"target": pPID})
	inv := p.expect("seat_invite", 2*time.Second)
	p.mustOK("seat_accept", msg{"target": inv["driver"]})
	if r := p.act("go", msg{"to": "station"}); r["ok"] == true {
		t.Fatal("khách không tự đi khi đang ngồi xe")
	}
	if r := d.act("dig_start", msg{"id": "x"}); r["ok"] == true {
		t.Fatal("không làm việc khác khi đang lái")
	}

	// Lái xe: khách đi theo ghế, không tự dịch chuyển.
	gz := Zones["garage"]
	d.walkTo(gz.CX, gz.CZ-3)
	time.Sleep(200 * time.Millisecond)
	if p.position().Dist(d.position()) > 1.5 {
		t.Fatalf("khách phải ngồi cạnh người lái: %v vs %v", p.position(), d.position())
	}
	// Cả xe chạy trên đường làng sang bến; khách ngồi theo.
	d.goZone("", "station")
	time.Sleep(200 * time.Millisecond)
	if e.areaOf(p) != "station" {
		t.Fatal("khách đi cùng xe sang khu mới")
	}
	// Khu cấm lái (trạm ấp): xe dừng ở mép khu, không chạy vào được.
	hz := Zones["hatchery"]
	d.goZone("", "village")
	d.walkTo(hz.Entry.X, hz.Entry.Z)
	for i := 0; i < 20; i++ {
		d.send(msg{"t": "in", "x": 0, "z": 1})
		time.Sleep(50 * time.Millisecond)
	}
	d.send(msg{"t": "in", "x": 0, "z": 0})
	if e.areaOf(d) == "hatchery" {
		t.Fatal("không lái xe vào trạm ấp")
	}
	if r := d.act("go", msg{"to": "hatchery"}); r["ok"] == true {
		t.Fatal("không còn cổng dịch chuyển giữa các khu")
	}
	d.goZone("", "station")
	// Đường liên trại: cả nhóm sang Ven Suối cùng một bản sao, vẫn ngồi đúng ghế.
	d.walkToItem("station", "duong_lien_trai")
	d.mustOK("drive_region", nil)
	wd := d.expect("welcome", 3*time.Second)
	wp := p.expect("welcome", 3*time.Second)
	if wd["room"].(map[string]any)["id"] != wp["room"].(map[string]any)["id"] || wd["zone"] != "vs_station" {
		t.Fatal("cả nhóm vào cùng một bản sao ở Bến Ven Suối")
	}
	found := false
	for _, x := range wp["players"].([]any) {
		if pp := x.(map[string]any); pp["name"] == "Gà Lái" && len(pp["seats"].([]any)) == 4 && pp["seats"].([]any)[1] != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("xe được dựng lại với khách trên ghế")
	}
}

func TestTradeBetweenPlayers(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "taa", "Gà A", func(a *Account) { a.Level = 3; a.Thoc = 100 })
	b, _ := e.seeded(t, "tbb", "Gà B", func(a *Account) { a.Level = 3; a.Worms["vang"] = 1 })
	a.expect("pjoin", 2*time.Second)
	pos := a.position()
	b.walkTo(pos.X+1, pos.Z)
	var bPID string
	for {
		m := a.expect("chat", 2*time.Second)
		_ = m
		break
	}
	for _, x := range e.roomOf(t).players {
		if x.acc.Char.Name == "Gà B" {
			bPID = x.PID
		}
	}
	a.mustOK("trade_invite", msg{"target": bPID})
	inv := b.expect("trade_invite", 2*time.Second)
	id := inv["id"]
	b.mustOK("trade_accept", msg{"id": id})
	a.mustOK("trade_offer", msg{"id": id, "i": 80})
	if r := b.act("trade_offer", msg{"id": id, "offer": map[string]int{"vang": 2}}); r["ok"] == true {
		t.Fatal("không đưa ra quá số đang có")
	}
	b.mustOK("trade_offer", msg{"id": id, "offer": map[string]int{"vang": 1}})
	a.mustOK("trade_lock", msg{"id": id})
	b.mustOK("trade_lock", msg{"id": id})
	if r := a.act("trade_offer", msg{"id": id, "i": 70}); r["ok"] != true {
		t.Fatal("sửa đề nghị")
	}
	if r := a.act("trade_confirm", msg{"id": id}); r["ok"] == true {
		t.Fatal("sửa đề nghị thì phải khoá lại từ đầu")
	}
	a.mustOK("trade_lock", msg{"id": id})
	b.mustOK("trade_lock", msg{"id": id})
	a.mustOK("trade_confirm", msg{"id": id})
	b.mustOK("trade_confirm", msg{"id": id})
	time.Sleep(100 * time.Millisecond)
	a.send(msg{"t": "ping"})
	r := a.mustOK("title", msg{"id": ""})
	if r["me"].(map[string]any)["thoc"].(float64) != 30 || r["me"].(map[string]any)["worms"].(map[string]any)["vang"].(float64) != 1 {
		t.Fatalf("A sau giao dịch: %v", r["me"])
	}
	rb := b.mustOK("title", msg{"id": ""})
	if rb["me"].(map[string]any)["thoc"].(float64) != 70 {
		t.Fatal("B nhận 70 thóc")
	}
}

func (e *env) roomOf(t *testing.T) *Room {
	e.hub.mu.Lock()
	defer e.hub.mu.Unlock()
	return e.hub.rooms[0]
}

func TestContractBuildFlow(t *testing.T) {
	e := newEnv(t, t.TempDir())
	o, oid := e.seeded(t, "chu", "Gà Chủ", func(a *Account) { a.Level = 3; a.Thoc = 1000; a.Items["go"], a.Items["dinh"] = 20, 10 })
	home, _ := e.store.Homes.Get(oid, "Gà Chủ")
	home.Tier = 1
	e.store.Homes.Save(home)
	w, _ := e.seeded(t, "tho", "Gà Thợ", func(a *Account) { a.Level = 7 })

	e.warp(o, "neighborhood")
	e.warp(w, "neighborhood")
	o.send(msg{"t": "visit", "owner": "me"})
	o.expect("welcome", 2*time.Second)
	if r := o.act("contract_create", msg{"id": "luc", "i": 300}); r["ok"] == true {
		t.Fatal("phải đứng cạnh Bò Mộc")
	}
	cow := home.Zone().Item("bo_moc")
	o.walkTo(cow.X, cow.Z+0.8)
	o.mustOK("contract_create", msg{"id": "luc", "i": 300})
	if o.num("thoc") != 700 {
		t.Fatal("giữ 300 thóc ký quỹ")
	}
	if r := o.act("home_build", msg{"id": "do"}); r["ok"] == true {
		t.Fatal("đang có hợp đồng thì không thuê Bò Mộc xây")
	}

	w.send(msg{"t": "contracts"})
	list := w.expect("contracts", 2*time.Second)["contracts"].([]any)
	kid := list[0].(map[string]any)["id"].(string)
	w.mustOK("contract_take", msg{"id": kid})
	w.send(msg{"t": "visit", "owner": oid})
	w.expect("welcome", 2*time.Second)
	cx, cz := coopCell(LandTiers[1])
	w.walkTo(float64(cx)+1.5, float64(cz))
	for i := 0; i < e.hub.bal.ContractWork; i++ {
		w.mustOK("contract_work", msg{"id": kid})
		time.Sleep(WorkGap + 100*time.Millisecond)
	}
	time.Sleep(1200 * time.Millisecond) // phòng cộng công mỗi giây
	w.send(msg{"t": "contracts"})
	w.expect("contracts", 2*time.Second)
	if r := w.mustOK("title", msg{"id": ""}); r["me"].(map[string]any)["thoc"].(float64) != 300 || r["me"].(map[string]any)["reputation"].(float64) != 1 {
		t.Fatalf("thợ nhận 300 thóc và +1 uy tín: %v", r["me"])
	}
	if home.Coop == nil || home.Coop.State != "done" || home.Coop.Roof != "luc" {
		t.Fatal("máy chủ tự nghiệm thu: chuồng mái xanh lá đã xong")
	}
}

func TestGuildFlow(t *testing.T) {
	e := newEnv(t, t.TempDir())
	o, _ := e.seeded(t, "hoitruong", "Gà Hội", func(a *Account) { a.Level = 10; a.Thoc = 5000 })
	m, mid := e.seeded(t, "thanhvien", "Gà Thành", func(a *Account) { a.Level = 2; a.Thoc = 300 })
	if r := m.act("guild_create", msg{"text": "Hội Nhỏ"}); r["ok"] == true {
		t.Fatal("lập hội cần cấp 10")
	}
	o.mustOK("guild_create", msg{"text": "Hội Thóc Vàng", "id": "🌾", "mode": "#3f86c9"})
	if o.num("thoc") != 5000-GuildCost {
		t.Fatal("phí lập hội")
	}
	o.send(msg{"t": "guild"})
	gid := o.expect("guild", 2*time.Second)["guild"].(map[string]any)["id"].(string)
	m.send(msg{"t": "guild_join", "id": gid})
	m.toastHas("chờ")
	o.send(msg{"t": "guild_approve", "id": mid})
	o.expect("guild", 2*time.Second)
	// Chat hội xuyên phòng: thành viên ở chuồng riêng vẫn nhận.
	e.warp(m, "neighborhood")
	m.send(msg{"t": "visit", "owner": "me"})
	m.expect("welcome", 2*time.Second)
	o.send(msg{"t": "guild_chat", "text": "họp hội tối nay"})
	if c := m.expect("guild_chat", 2*time.Second); c["text"] != "họp hội tối nay" {
		t.Fatal("chat hội xuyên phòng")
	}
	// Quỹ chỉ chi cho công trình định nghĩa; thành viên thường không chi được.
	m.mustOK("guild_give", msg{"i": 200})
	o.mustOK("guild_give", msg{"i": 1900})
	m.send(msg{"t": "guild_project", "id": "co_hoi"})
	m.toastHas("Vai trò")
	o.send(msg{"t": "guild_project", "id": "co_hoi"})
	g := o.expect("guild", 2*time.Second)["guild"].(map[string]any)
	if g["fund"].(float64) != 100 || g["projects"].(map[string]any)["co_hoi"] != true {
		t.Fatalf("xây cờ hội từ quỹ: %v", g["fund"])
	}
	// Chuyển quyền cần người nhận xác nhận.
	o.send(msg{"t": "guild_transfer", "id": mid})
	o.expect("guild", 2*time.Second)
	m.send(msg{"t": "guild_transfer_accept"})
	if g := m.expect("guild", 2*time.Second)["guild"].(map[string]any); g["owner"] != mid {
		t.Fatal("chuyển quyền chủ hội")
	}
}

func TestLeaveStationCancelsBooking(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.BusWait = time.Minute })
	a, _ := e.seeded(t, "rongchoi", "Gà Rong", func(a *Account) { a.Level = 3 })
	a.goZone("ra_ben", "station")
	a.walkToItem("station", "tram_bus")
	a.mustOK("book", nil)
	if a.num("tripsLeft") != MaxBusTokens-1 {
		t.Fatal("giữ một lượt miễn phí")
	}
	// Rời bến trước giờ xe chạy: vé tự huỷ, lượt trả lại đúng một lần.
	a.goZone("", "village") // đi bộ ra khỏi bến
	a.toastHas("rời bến")
	r := a.mustOK("title", msg{"id": ""})
	if r["me"].(map[string]any)["tripsLeft"].(float64) != MaxBusTokens || r["me"].(map[string]any)["booking"] != nil {
		t.Fatal("rời bến trước giờ chạy phải trả lượt")
	}
}

func TestGuildHall(t *testing.T) {
	e := newEnv(t, t.TempDir())
	o, _ := e.seeded(t, "chuhoi", "Gà Hội", func(a *Account) { a.Level = 10; a.Thoc = 2000 })
	x, _ := e.seeded(t, "nguoila", "Gà Lạ", func(a *Account) {})
	o.mustOK("guild_create", msg{"text": "Hội Rơm"})
	o.send(msg{"t": "guild_hall"})
	o.toastHas("Xóm Rơm Vàng")
	e.warp(o, "neighborhood")
	e.warp(x, "neighborhood")
	x.send(msg{"t": "guild_hall"})
	x.toastHas("chưa ở trong hội")
	o.send(msg{"t": "guild_hall"})
	w := o.expect("welcome", 2*time.Second)
	if w["zone"] != "guildhall" || w["guildhall"] == nil || w["room"].(map[string]any)["kind"] != "guild" {
		t.Fatalf("vào Nhà Hội: %v", w["room"])
	}
	o.mustOK("go", msg{"to": "lobby"})
	if w := o.expect("welcome", 2*time.Second); w["zone"] != "neighborhood" {
		t.Fatal("rời Nhà Hội ra Xóm Rơm Vàng")
	}
	time.Sleep(100 * time.Millisecond)
	e.hub.mu.Lock()
	open := len(e.hub.homes)
	e.hub.mu.Unlock()
	if open != 0 {
		t.Fatal("Nhà Hội trống phải đóng")
	}
}
