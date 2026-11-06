package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Hub giữ danh bạ chuồng công cộng và phiên đang điều khiển của mỗi tài khoản.
// Quy tắc khoá: goroutine của Room không bao giờ lấy hub.mu, nên hub được phép chờ Room.

type Hub struct {
	store *Store
	bal   Balance

	mu          sync.Mutex
	rooms       []*Room
	sessions    map[string]*Session // account ID → phiên đang điều khiển nhân vật
	roomSeq     int
	homes       map[string]*Room // chủ nhà → phòng nhà đang mở
	homeInvites map[string]homeInvite
	lastPublic  map[*Session]string

	staff       map[string]string // username → mod | econ | admin
	devWarp     bool              // -devwarp: cho phép "go" nhảy khu, chỉ để chụp ảnh QA
	maintenance bool
	maintText   string
	festival    *Festival
	weekly      *Weekly

	accLocks sync.Map // account ID → *sync.Mutex: tuần tự hoá Hello của cùng một tài khoản
}

// accountLock trả về khoá riêng của một tài khoản. Hai "hello" cùng lúc phải chạy lần lượt, nếu không
// cả hai cùng thấy chưa có phiên cũ và một tài khoản có hai phiên điều khiển (nhân đồ, sập vì ghi map đồng thời).
func (h *Hub) accountLock(id string) *sync.Mutex {
	l, _ := h.accLocks.LoadOrStore(id, &sync.Mutex{})
	return l.(*sync.Mutex)
}

func NewHub(store *Store, bal Balance) *Hub {
	fest, err := OpenFestival(store.dir)
	if err != nil {
		log.Printf("festival: %v", err)
		fest = &Festival{st: FestivalState{Name: "Hội đèn lồng mùa thu", Goal: 5000}, path: filepath.Join(store.dir, "festival.json")}
	}
	h := &Hub{store: store, bal: bal, sessions: map[string]*Session{}, homes: map[string]*Room{},
		homeInvites: map[string]homeInvite{}, lastPublic: map[*Session]string{}, staff: map[string]string{}, festival: fest,
		weekly: OpenWeekly(store.dir)}
	go h.contractJanitor()
	return h
}

type Session struct {
	hub     *Hub
	conn    *websocket.Conn
	send    chan []byte
	acc     *Account
	room    atomic.Pointer[Room]
	closed  atomic.Bool
	sendMu  sync.Mutex
	kicked  atomic.Bool // bị thay phiên/kick: không giữ chỗ nối lại
	arrive  *arrival    // nơi xuất hiện ở phòng kế tiếp (đặt trước khi add, đọc trong goroutine phòng)
	inBurst float64
	inAt    time.Time
	// msgBurst/msgAt: xô rò cho mọi message khác "in"/"ping" (chỉ goroutine đọc của phiên chạm vào).
	msgBurst float64
	msgAt    time.Time
	// moveMu tuần tự hoá mọi lần đổi phòng của phiên (đổi kênh, sang vùng, vào/ra nhà, Nhà Hội, bị thay phiên).
	// Thiếu nó, hai lần chuyển chạy song song có thể thêm cùng một phiên vào phòng hai lần (gà "ma") hoặc vào hai phòng.
	moveMu sync.Mutex
}

func (s *Session) Send(v any) {
	if s.closed.Load() {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("marshal: %v", err)
		return
	}
	s.sendRaw(b)
}

var sendDrops atomic.Int64

// contractJanitor đối soát hợp đồng quá hạn: hoàn ký quỹ đúng một lần, báo chủ qua thư (mục 9.2).
func (h *Hub) contractJanitor() {
	for range time.Tick(2 * time.Second) {
		for _, k := range h.store.Contracts.Expire(time.Now()) {
			if h.store.Social.AddMail(k.Owner, "build", fmt.Sprintf("⌛ Hợp đồng xây chuồng quá hạn: đã hoàn ký quỹ %d thóc.", k.Price)) {
				h.mailPing(k.Owner)
			}
		}
	}
}

func (s *Session) Close() {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	s.closeLocked()
}

func (s *Session) closeLocked() {
	if s.closed.CompareAndSwap(false, true) {
		close(s.send)
	}
}

var upgrader = websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 8192, CheckOrigin: func(r *http.Request) bool { return true }}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s := &Session{hub: h, conn: conn, send: make(chan []byte, 256)}
	go s.writePump()
	s.readPump()
}

func (s *Session) writePump() {
	ping := time.NewTicker(25 * time.Second)
	defer func() { ping.Stop(); s.conn.Close() }()
	for {
		select {
		case b, ok := <-s.send:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				s.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, b); err != nil {
				return
			}
		case <-ping.C:
			s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type inMsg struct {
	throttled bool // do allowMsg đặt, không đến từ JSON: act vượt giới hạn tần suất
	T         string  `json:"t"`
	Token string  `json:"token"`
	X     float64 `json:"x"`
	Z     float64 `json:"z"`
	Text  string  `json:"text"`
	E     string  `json:"e"`
	Room  string  `json:"room"`
	C     float64 `json:"c"`
	// tạo nhân vật
	Name  string `json:"name"`
	Color string `json:"color"`
	Comb  string `json:"comb"`
	Shirt string `json:"shirt"`
	Sex   string `json:"sex"`
	// hành động
	RID     string       `json:"rid"`
	A       string       `json:"a"`
	ID      string       `json:"id"`
	To      string       `json:"to"`
	Slot    string       `json:"slot"`
	Eat     bool         `json:"eat"`
	I       int          `json:"i"`
	Elapsed float64      `json:"elapsed"`
	Target  string       `json:"target"`
	Run     bool         `json:"run"` // gói "in": đang giữ nút chạy
	Ev      [][2]float64 `json:"ev"`  // dig_finish: [giây từ lúc giằng co, 1 giữ / 0 thả]
	Mode    string       `json:"mode"`
	Side    string       `json:"side"`
	// xã hội & nhà
	Owner    string         `json:"owner"`
	On       bool           `json:"on"`
	Presence bool           `json:"presence"`
	AllowPK  bool           `json:"allowPK"`
	Reason   string         `json:"reason"`
	Tail     string         `json:"tail"`
	Eyes     string         `json:"eyes"`
	Resume   bool           `json:"resume"`
	Offer    map[string]int `json:"offer"`
	Items    map[string]int `json:"items"` // trade_offer: vật phẩm kho đưa ra (giun ở Offer)
}

func (s *Session) readPump() {
	defer s.hub.Leave(s)
	s.conn.SetReadLimit(16384) // đủ cho 400 sự kiện giằng co hợp lệ (lỗi thấp L3)
	s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	s.conn.SetPongHandler(func(string) error { s.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, data, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		s.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		var m inMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		if !s.handleSafe(&m) {
			return // panic khi xử lý: ngắt phiên này (Leave qua defer) thay vì sập cả máy chủ
		}
	}
}

// handleSafe bắt panic trong handler chạy trên goroutine đọc của phiên.
func (s *Session) handleSafe(m *inMsg) (ok bool) {
	defer func() {
		if e := recover(); e != nil {
			logPanic("session "+m.T, e)
			ok = false
		}
	}()
	s.handle(m)
	return true
}

// Giới hạn tần suất cho mọi message ngoài "in"/"ping": mỗi act thành công ghi cả tài khoản xuống đĩa trên
// goroutine phòng, mỗi lệnh xã hội ghi file; spam không giới hạn sẽ làm chậm tick của cả phòng.
const (
	MsgRatePerSec = 20.0
	MsgBurst      = 40.0
)

// Kết quả rateMsg.
const (
	msgAllow    = iota // xử lý bình thường
	msgThrottle        // vượt MsgBurst: act nhận "res" lỗi, message khác bị bỏ
	msgDrop            // vượt 2×MsgBurst: bỏ hẳn, không đẩy gì vào hàng đợi phòng
)

// rateMsg: xô rò MsgRatePerSec/giây. Message bị từ chối vẫn tính vào xô (tối đa 2×MsgBurst+1) để kẻ spam
// không trả lời được thì bị bỏ hẳn: nếu không, act bị chặn vẫn đầy hộp thư của phòng và làm nghẽn người khác.
func (s *Session) rateMsg(now time.Time) int {
	s.msgBurst = max(0, s.msgBurst-now.Sub(s.msgAt).Seconds()*MsgRatePerSec)
	s.msgAt = now
	s.msgBurst = min(s.msgBurst+1, 2*MsgBurst+1)
	switch {
	case s.msgBurst <= MsgBurst:
		return msgAllow
	case s.msgBurst <= 2*MsgBurst:
		return msgThrottle
	}
	return msgDrop
}

func (s *Session) handle(m *inMsg) {
	if s.acc == nil {
		if m.T == "hello" {
			s.hub.Hello(s, m.Token, m.Resume)
		}
		return
	}
	if m.T != "in" && m.T != "ping" {
		switch s.rateMsg(time.Now()) {
		case msgThrottle:
			if m.T != "act" {
				return
			}
			m.throttled = true // vẫn chuyển vào phòng để trả "res" lỗi, client không phải chờ mãi
		case msgDrop:
			return
		}
	}
	switch m.T {
	case "create_char":
		s.hub.CreateChar(s, m)
	case "rooms":
		s.Send(map[string]any{"t": "rooms", "rooms": s.hub.RoomList(s)})
	case "switch_room":
		s.hub.Switch(s, m.Room)
	case "ping":
		s.Send(map[string]any{"t": "pong", "c": m.C})
	case "guild", "guild_list", "guild_join", "guild_approve", "guild_deny", "guild_invite", "guild_leave", "guild_kick",
		"guild_role", "guild_transfer", "guild_transfer_accept", "guild_claim", "guild_mode", "guild_post", "guild_project", "guild_chat", "guild_hall":
		if s.acc.Char != nil {
			s.hub.guild(s, m)
		}
	case "social", "mail", "contracts", "friend_add", "friend_respond", "friend_remove", "block", "report", "dm", "dm_history",
		"privacy", "invite_room", "visit", "leave_home", "home_invite", "homes":
		if s.acc.Char != nil {
			s.hub.social(s, m)
		}
	case "in":
		// Giới hạn tần suất input: tối đa ~30 gói/giây, phần dư bỏ qua.
		now := time.Now()
		s.inBurst = max(0, s.inBurst-now.Sub(s.inAt).Seconds()*30)
		s.inAt = now
		if s.inBurst > 30 {
			return
		}
		s.inBurst++
		fallthrough
	default:
		if r := s.room.Load(); r != nil {
			mm := *m
			r.Do(func() { r.handle(s, &mm) })
		}
	}
}

// Hello xác thực token và giành quyền điều khiển nhân vật (một phiên/tài khoản).
// Nếu phiên cũ vừa mất kết nối và còn trong thời gian giữ chỗ thì nối lại phiên đó (mục 6.3).
func (h *Hub) Hello(s *Session, token string, resume bool) {
	id, ok := h.store.AccountIDForToken(token)
	if !ok {
		s.Send(map[string]any{"t": "auth_fail", "err": "Phiên đăng nhập hết hạn, hãy đăng nhập lại."})
		return
	}
	lk := h.accountLock(id)
	lk.Lock()
	defer lk.Unlock()
	acc, err := h.store.Account(id)
	if err != nil {
		s.Send(map[string]any{"t": "auth_fail", "err": "Không đọc được tài khoản."})
		return
	}
	if until, reason, banned := h.store.Mod.Banned(id); banned {
		s.Send(map[string]any{"t": "auth_fail", "err": fmt.Sprintf("Tài khoản bị tạm khoá đến %s. Lý do: %s", until.In(vnZone).Format("15:04 02/01/2006"), reason)})
		return
	}
	h.mu.Lock()
	maint, text := h.maintenance, h.maintText
	_, isStaff := h.staff[acc.Username]
	old := h.sessions[id]
	h.mu.Unlock()
	if maint && !isStaff && (old == nil || !old.closed.Load()) {
		s.Send(map[string]any{"t": "maintenance", "text": text})
		return
	}
	s.acc = acc
	h.store.Guilds.Touch(id) // M8: chủ hội đăng nhập hằng ngày không bị coi là vắng mặt
	if old != nil && old != s && old.closed.Load() && !old.kicked.Load() {
		if r := old.room.Load(); r != nil {
			resumed := false
			r.DoWait(func() { resumed = r.rebind(old, s, resume) })
			if resumed {
				h.mu.Lock()
				h.sessions[id] = s
				h.lastPublic[s] = h.lastPublic[old]
				delete(h.lastPublic, old)
				h.mu.Unlock()
				old.room.Store(nil)
				s.room.Store(r)
				return
			}
		}
	}
	h.mu.Lock()
	h.sessions[id] = s
	h.mu.Unlock()
	if old != nil && old != s {
		old.kicked.Store(true)
		old.Send(map[string]any{"t": "kicked", "reason": "Tài khoản vừa vào game ở nơi khác."})
		old.moveMu.Lock() // chờ lần đổi phòng đang dở của phiên cũ xong, rồi mới gỡ nó khỏi phòng hiện tại
		if r := old.room.Load(); r != nil {
			r.DoWait(func() { r.remove(old) }) // phiên cũ mất quyền trước khi phiên mới hoạt động
		}
		old.room.Store(nil)
		old.moveMu.Unlock()
		time.AfterFunc(300*time.Millisecond, old.Close)
	}
	if acc.Char != nil {
		h.store.Social.Ensure(acc.ID, acc.Char.Name)
	}
	if acc.Char == nil {
		s.Send(map[string]any{"t": "need_char", "catalog": catalog(h.bal)})
		return
	}
	if bk := acc.Booking; bk != nil && bk.State == "departed" && Regions[bk.To] != nil {
		h.placeIn(s, "", bk.To, &arrival{Zone: Regions[bk.To].Station, Trip: bk.Trip})
		return
	}
	h.place(s, "")
}

func (h *Hub) CreateChar(s *Session, m *inMsg) {
	if s.acc.Char != nil {
		return
	}
	name, err := ValidateName(m.Name)
	if m.Tail == "" {
		m.Tail = TailKinds[0]
	}
	if m.Eyes == "" {
		m.Eyes = EyeKinds[0]
	}
	if err == nil && (!contains(FeatherColors, m.Color) || !contains(CombKinds, m.Comb) || !contains(StartShirts, m.Shirt) ||
		!contains(ProfileSexes, m.Sex) || !contains(TailKinds, m.Tail) || !contains(EyeKinds, m.Eyes)) {
		err = fmt.Errorf("Lựa chọn ngoại hình không hợp lệ.")
	}
	if err != nil {
		s.Send(map[string]any{"t": "char_err", "err": err.Error()})
		return
	}
	s.acc.Char = &Character{Name: name, Color: m.Color, Comb: m.Comb, Shirt: m.Shirt, Sex: m.Sex, Tail: m.Tail, Eyes: m.Eyes}
	if err := h.store.Save(s.acc); err != nil {
		s.acc.Char = nil
		s.Send(map[string]any{"t": "char_err", "err": "Chưa lưu được nhân vật, thử lại."})
		return
	}
	if _, err := h.store.Social.Ensure(s.acc.ID, name); err != nil {
		log.Printf("social ensure: %v", err)
	}
	h.place(s, "")
}

// arrival mô tả nơi xuất hiện khi chuyển vùng (xe buýt, xe riêng) thay cho sảnh mặc định.
type arrival struct {
	Zone string
	Trip string
	Ride *rideSpec
	Pos  *Vec // đổi kênh: đứng lại đúng chỗ cũ trên bản đồ
}

// place đưa phiên vào sảnh của một chuồng còn chỗ (mục 6.3): ưu tiên chuồng đang có người, rồi mới mở chuồng mới.
func (h *Hub) place(s *Session, prefer string) {
	region := RegionHypex
	h.mu.Lock()
	for _, r := range h.rooms {
		if r.ID == prefer {
			region = r.Region
		}
	}
	h.mu.Unlock()
	h.placeIn(s, prefer, region, nil)
}

func (h *Hub) placeIn(s *Session, prefer, region string, arr *arrival) {
	h.mu.Lock()
	target := h.pickRoomLocked(prefer, region, 1)
	target.reserve()
	h.lastPublic[s] = target.ID
	h.mu.Unlock()
	s.arrive = arr
	s.room.Store(target)
	target.DoWait(func() { target.add(s) })
}

// pickRoomLocked chọn chuồng trong vùng còn đủ seats chỗ (cả nhóm đi cùng xe vào cùng một bản sao).
func (h *Hub) pickRoomLocked(prefer, region string, seats int) *Room {
	for _, r := range h.rooms {
		if r.ID == prefer && r.Region == region && r.Count()+seats <= RoomCapacity {
			return r
		}
	}
	var target *Room
	best := -1
	for _, r := range h.rooms {
		if c := r.Count(); r.Region == region && c+seats <= RoomCapacity && c > best {
			best, target = c, r
		}
	}
	if target == nil {
		target = h.newRoomLocked(region)
	}
	return target
}

func (h *Hub) newRoomLocked(region string) *Room {
	h.roomSeq++
	reg := Regions[region]
	n := 0
	for _, r := range h.rooms {
		if r.Region == region {
			n++
		}
	}
	theme := reg.Themes[n%len(reg.Themes)]
	num := n/len(reg.Themes) + 1
	r := NewRegionRoom(h, fmt.Sprintf("c%d", h.roomSeq), fmt.Sprintf("%s · %02d", theme, num), region)
	h.rooms = append(h.rooms, r)
	go r.Run()
	return r
}

// emptyOrNewRoomLocked phục vụ "chuồng mới": dùng lại một chuồng đang trống của vùng, chỉ mở chuồng mới khi
// không còn chuồng trống. Phòng công khai không bao giờ đóng, nên nếu luôn mở mới thì một client lặp lệnh
// sẽ đẻ vô hạn goroutine 20 tick/giây. Cách này giữ số phòng ≤ số phòng có người + 1.
func (h *Hub) emptyOrNewRoomLocked(region string, cur *Room) *Room {
	for _, r := range h.rooms {
		if r != cur && r.Region == region && r.Count() == 0 {
			return r
		}
	}
	return h.newRoomLocked(region)
}

// Switch đổi sang bản sao chuồng khác trong cùng vùng, không tốn vé (mục 8.5).
func (h *Hub) Switch(s *Session, roomID string) {
	s.moveMu.Lock()
	defer s.moveMu.Unlock()
	cur := s.room.Load()
	if cur == nil || cur.ID == roomID {
		return
	}
	region := cur.Region
	if cur.Kind == "home" {
		region = RegionHypex
	}
	h.mu.Lock()
	var target *Room
	for _, r := range h.rooms {
		if r.ID == roomID && r.Region == region {
			target = r
		}
	}
	if target == nil && roomID == "new" {
		target = h.emptyOrNewRoomLocked(region, cur)
	}
	if target == nil || target.Count() >= RoomCapacity {
		h.mu.Unlock()
		s.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Chuồng này đã đầy hoặc không ở cùng vùng. Chọn chuồng khác nhé."})
		return
	}
	target.reserve()
	h.lastPublic[s] = target.ID
	h.mu.Unlock()

	ok := true
	cur.DoWait(func() { ok = cur.canLeave(s) })
	if !ok {
		target.unreserve()
		s.Send(map[string]any{"t": "toast", "kind": "warn", "text": "Đang trong trận PK hoặc đang trên xe, chưa đổi chuồng được."})
		return
	}
	var arr *arrival
	cur.DoWait(func() {
		if p := cur.players[s]; p != nil && cur.world != nil && cur.Region == target.Region {
			pos := p.pos
			arr = &arrival{Zone: p.zone, Pos: &pos}
		}
		cur.remove(s)
	})
	s.arrive = arr
	s.room.Store(target)
	target.DoWait(func() { target.add(s) })
}

// TransferRegion đưa một nhóm (1 người hoặc cả xe) sang vùng khác, giữ đủ chỗ trong cùng một bản sao trước khi chuyển.
func (h *Hub) TransferRegion(group []*Session, region string, arrs []*arrival) {
	if len(group) == 0 {
		return
	}
	h.mu.Lock()
	target := h.pickRoomLocked("", region, len(group))
	for range group {
		target.reserve()
	}
	for _, s := range group {
		h.lastPublic[s] = target.ID
	}
	h.mu.Unlock()
	for i, s := range group {
		h.transferOne(s, target, arrs[i])
	}
}

// transferOne chuyển một phiên sang phòng đích đã giữ chỗ, dưới khoá đổi phòng của phiên.
func (h *Hub) transferOne(s *Session, target *Room, arr *arrival) {
	s.moveMu.Lock()
	defer s.moveMu.Unlock()
	if cur := s.room.Load(); cur != nil {
		cur.DoWait(func() { cur.remove(s) })
	}
	if s.closed.Load() || s.kicked.Load() {
		s.room.Store(nil)
		target.unreserve()
		h.mu.Lock()
		delete(h.lastPublic, s) // TransferRegion vừa ghi mục này; phiên đã đóng thì dropSession có thể đã chạy xong
		h.mu.Unlock()
		return
	}
	s.arrive = arr
	s.room.Store(target)
	target.DoWait(func() { target.add(s) })
}

func (h *Hub) Leave(s *Session) {
	// Mất kết nối: giữ chỗ trong phòng một lúc để nối lại, nhất là khi đang đấu PK (mục 10.2).
	s.moveMu.Lock() // đợi lần đổi phòng đang dở xong để đánh dấu mất kết nối đúng phòng
	defer s.moveMu.Unlock()
	if r := s.room.Load(); r != nil && !s.kicked.Load() && s.acc != nil && s.acc.Char != nil {
		s.Close()
		r.Do(func() { r.markGone(s) })
		return
	}
	h.dropSession(s)
}

// dropSession dọn phiên khỏi hub và phòng (sau khi hết giờ giữ chỗ hoặc bị kick).
func (h *Hub) dropSession(s *Session) {
	h.mu.Lock()
	if s.acc != nil && h.sessions[s.acc.ID] == s {
		delete(h.sessions, s.acc.ID)
	}
	delete(h.lastPublic, s)
	h.mu.Unlock()
	if r := s.room.Swap(nil); r != nil {
		r.DoWait(func() { r.remove(s) })
	}
	s.Close()
}

type roomInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
	Cap   int    `json:"cap"`
	Here  bool   `json:"here"`
}

func (h *Hub) RoomList(s *Session) []roomInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	cur := s.room.Load()
	region := RegionHypex
	if cur != nil && cur.Kind == "public" {
		region = cur.Region
	}
	var out []roomInfo
	for _, r := range h.rooms {
		if r.Region == region {
			out = append(out, roomInfo{r.ID, r.Name, r.Count(), RoomCapacity, r == cur})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ---------- HTTP ----------

func (h *Hub) authHandler(register bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		var body struct{ Username, Password string }
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]any{"err": "Dữ liệu không hợp lệ."})
			return
		}
		var token, rec string
		var err error
		if register {
			token, rec, err = h.store.Register(body.Username, body.Password)
		} else {
			token, err = h.store.Login(body.Username, body.Password)
		}
		if err != nil {
			writeJSON(w, 400, map[string]any{"err": err.Error()})
			return
		}
		out := map[string]any{"token": token}
		if rec != "" {
			out["recovery"] = rec // chỉ hiện một lần lúc đăng ký
		}
		writeJSON(w, 200, out)
	}
}

// Routes đăng ký toàn bộ API của máy chủ (dùng chung cho main và kiểm thử).
func (h *Hub) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", h.ServeWS)
	mux.HandleFunc("/api/register", h.authHandler(true))
	mux.HandleFunc("/api/login", h.authHandler(false))
	mux.HandleFunc("/api/status", h.statusHandler)
	mux.HandleFunc("/api/recover", h.recoverHandler)
	h.RegisterAdmin(mux)
	return mux
}

// recoverHandler đặt lại mật khẩu bằng mã khôi phục và đá mọi phiên đang chơi của tài khoản.
func (h *Hub) recoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var body struct{ Username, Code, Password string }
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048)).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"err": "Dữ liệu không hợp lệ."})
		return
	}
	token, rec, id, err := h.store.Recover(body.Username, body.Code, body.Password)
	if err != nil {
		writeJSON(w, 400, map[string]any{"err": err.Error()})
		return
	}
	h.kick(id, "Mật khẩu vừa được đặt lại. Hãy đăng nhập lại.")
	writeJSON(w, 200, map[string]any{"token": token, "recovery": rec})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func (h *Hub) statusHandler(w http.ResponseWriter, r *http.Request) {
	m := h.Metrics()
	writeJSON(w, 200, map[string]any{"online": m.Online, "rooms": m.Rooms, "balance": h.bal.Name, "metrics": m})
}

func catalog(b Balance) map[string]any {
	return map[string]any{
		"worms": AllWorms, "season": CurrentSeason(time.Now()), "seasons": Seasons, "foods": Foods, "vehicles": Vehicles, "eggMilestones": EggMilestones, "busSeats": BusSeats,
		"freeTrips": FreeTrips, "busFare": BusFare, "companionMax": CompanionMax, "guildMax": GuildMax, "guildLevel": GuildLevel,
		"guildCost": GuildCost, "guildProjects": GuildProjects, "tradeLevel": TradeLevel, "contractLevel": ContractLevel, "regions": regionInfo(), "cosmetics": Cosmetics, "slots": Slots,
		"colors": FeatherColors, "combs": CombKinds, "shirts": StartShirts, "sexes": ProfileSexes,
		"tutorial": TutorialSteps, "balance": b, "levelCap": LevelCap, "roomCap": RoomCapacity,
		"tails": TailKinds, "eyes": EyeKinds, "decors": Decors, "landTiers": LandTiers, "coopCost": CoopCost,
		"roofs": RoofColors, "homeCap": HomeCapacity, "homeObjLimit": HomeObjLimit, "emotes": Emotes,
		"titles": TitleDefs, "festivalAmounts": FestivalAmounts, "bagCap": BagCapacity, "reconnectSec": b.ReconnectGrace.Seconds(),
		// PLAN_LOGIC_GAME
		"walkSpeed": WalkSpeed, "sprintMult": SprintMult, "staminaSec": StaminaSec, "tugTiers": TugTiers, "digXP": DigXP,
		"items": Items, "storageCap": StorageCap, "levelRewards": LevelRewards, "checkInRewards": CheckInRewards,
		"species": Species, "eggSpecies": EggSpecies, "eggKinds": EggKinds, "seeds": Seeds,
		"maxBusTokens": MaxBusTokens, "villageFare": VillageFare,
		"houseDesigns": HouseDesigns, "landPlaces": LandPlaces, "materialNames": MaterialNames, "comfort": Comfort,
		"weeklyKinds": WeeklyKinds, "gameDay": GameDay.Seconds(), "fishTable": FishTable,
	}
}
