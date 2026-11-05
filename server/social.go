package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Dữ liệu xã hội (mục 11) tách khỏi Account: bạn bè, lời mời, chặn, tin riêng, quyền riêng tư.
// Hai người ở hai phòng khác nhau cùng cần sửa nên mọi thao tác đi qua khoá chung của SocialStore,
// không phụ thuộc goroutine phòng đang giữ tài khoản.

type DM struct {
	ID   string    `json:"id"`
	From string    `json:"from"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

type Privacy struct {
	ShowPresence bool `json:"showPresence"` // cho bạn bè thấy mình đang ở chuồng nào
	AllowPK      bool `json:"allowPK"`      // nhận lời mời PK
	QuietMail    bool `json:"quietMail"`    // giảm thư: chỉ nhận thư công trình/hệ thống
}

// Thư Chim Chíp (mục 11): bạn ghé nhà, công trình xong, trứng nở, lời mời kết bạn.
type MailItem struct {
	ID   string    `json:"id"`
	Kind string    `json:"kind"`
	Text string    `json:"text"`
	At   time.Time `json:"at"`
	Read bool      `json:"read"`
}

const MailKeep = 40

type Social struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Code    string               `json:"code"`
	Friends map[string]time.Time `json:"friends"`
	ReqIn   map[string]time.Time `json:"reqIn"`
	ReqOut  map[string]time.Time `json:"reqOut"`
	Blocks  map[string]time.Time `json:"blocks"`
	DMs     map[string][]DM      `json:"dms"` // đối phương → tin gần nhất
	Unread  map[string]int       `json:"unread"`
	Privacy Privacy              `json:"privacy"`
	Mail    []MailItem           `json:"mail"`
}

const (
	DMKeep        = 60
	FriendLimit   = 100
	DMMaxLen      = 200
	ReportReasons = "spam|ngon_tu|ten|quay_roi|gian_lan"
	codeAlphabet  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

type SocialStore struct {
	dir     string
	mu      sync.Mutex
	cache   map[string]*Social
	codes   map[string]string // mã người chơi → account ID
	reports []Report
	repPath string
}

func OpenSocial(dir string) (*SocialStore, error) {
	sdir := filepath.Join(dir, "social")
	if err := os.MkdirAll(sdir, 0o755); err != nil {
		return nil, err
	}
	s := &SocialStore{dir: sdir, cache: map[string]*Social{}, codes: map[string]string{}}
	if b, err := os.ReadFile(filepath.Join(dir, "codes.json")); err == nil {
		json.Unmarshal(b, &s.codes)
	}
	s.repPath = filepath.Join(dir, "reports.json")
	if b, err := os.ReadFile(s.repPath); err == nil {
		json.Unmarshal(b, &s.reports)
	}
	return s, nil
}

func (s *SocialStore) getLocked(id string) *Social {
	if so, ok := s.cache[id]; ok {
		return so
	}
	so := &Social{ID: id}
	if !validAccountID(id) {
		// ID lạ từ client: trả bản rỗng, không đọc file, không cache; saveLocked sẽ từ chối ghi.
		so.Friends, so.ReqIn, so.ReqOut, so.Blocks = map[string]time.Time{}, map[string]time.Time{}, map[string]time.Time{}, map[string]time.Time{}
		so.DMs, so.Unread = map[string][]DM{}, map[string]int{}
		return so
	}
	if b, err := os.ReadFile(filepath.Join(s.dir, id+".json")); err == nil {
		json.Unmarshal(b, so)
	} else {
		so.Privacy = Privacy{ShowPresence: true, AllowPK: true}
	}
	for _, m := range []*map[string]time.Time{&so.Friends, &so.ReqIn, &so.ReqOut, &so.Blocks} {
		if *m == nil {
			*m = map[string]time.Time{}
		}
	}
	if so.DMs == nil {
		so.DMs = map[string][]DM{}
	}
	if so.Unread == nil {
		so.Unread = map[string]int{}
	}
	s.cache[id] = so
	return so
}

func (s *SocialStore) saveLocked(ids ...string) error {
	for _, id := range ids { // kiểm hết trước khi ghi, để không ghi dở một nửa cặp
		if !validAccountID(id) {
			return ErrNotFound
		}
	}
	for _, id := range ids {
		if err := writeAtomic(filepath.Join(s.dir, id+".json"), s.cache[id]); err != nil {
			return err
		}
	}
	return nil
}

// Ensure tạo mã người chơi duy nhất khi nhân vật có tên (mục 5: tìm bạn khi trùng tên).
func (s *SocialStore) Ensure(id, name string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	so := s.getLocked(id)
	so.Name = name
	if so.Code == "" {
		for {
			b := []byte(randHex(5))
			code := make([]byte, 6)
			for i := range code {
				code[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
			}
			c := string(code)
			if _, taken := s.codes[c]; !taken {
				so.Code = c
				s.codes[c] = id
				break
			}
		}
		if err := writeAtomic(filepath.Join(filepath.Dir(s.dir), "codes.json"), s.codes); err != nil {
			return "", err
		}
	}
	return so.Code, s.saveLocked(id)
}

func (s *SocialStore) Lookup(code string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.codes[strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(code, "#")))]
	return id, ok
}

// Blocked: a chặn b hoặc b chặn a — mọi kênh tương tác giữa hai người đều đóng (mục 11).
func (s *SocialStore) Blocked(a, b string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, x := s.getLocked(a).Blocks[b]
	_, y := s.getLocked(b).Blocks[a]
	return x || y
}

func (s *SocialStore) BlockedBy(owner, other string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, x := s.getLocked(owner).Blocks[other]
	return x
}

func (s *SocialStore) AreFriends(a, b string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.getLocked(a).Friends[b]
	return ok
}

func (s *SocialStore) Privacy(id string) Privacy {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id).Privacy
}

func (s *SocialStore) SetPrivacy(id string, p Privacy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getLocked(id).Privacy = p
	return s.saveLocked(id)
}

var (
	ErrSelf     = errors.New("Không thể tự kết bạn với chính mình.")
	ErrBlocked  = errors.New("Không thể tương tác với người chơi này.")
	ErrNotFound = errors.New("Không tìm thấy người chơi với mã này.")
)

// Request gửi lời mời kết bạn; nếu bên kia đã mời mình thì thành bạn luôn. Trả về true khi đã là bạn.
func (s *SocialStore) Request(from, to string) (bool, error) {
	if from == to {
		return false, ErrSelf
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, b := s.getLocked(from), s.getLocked(to)
	if _, x := a.Blocks[to]; x {
		return false, ErrBlocked
	}
	if _, y := b.Blocks[from]; y {
		return false, ErrBlocked
	}
	if _, ok := a.Friends[to]; ok {
		return true, nil
	}
	if len(a.Friends) >= FriendLimit || len(b.Friends) >= FriendLimit {
		return false, errors.New("Danh sách bạn đã đầy.")
	}
	now := time.Now()
	if _, ok := a.ReqIn[to]; ok {
		delete(a.ReqIn, to)
		delete(b.ReqOut, from)
		a.Friends[to], b.Friends[from] = now, now
		return true, s.saveLocked(from, to)
	}
	a.ReqOut[to] = now
	b.ReqIn[from] = now
	return false, s.saveLocked(from, to)
}

func (s *SocialStore) Respond(me, other string, accept bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, b := s.getLocked(me), s.getLocked(other)
	if _, ok := a.ReqIn[other]; !ok {
		return errors.New("Lời mời kết bạn không còn.")
	}
	delete(a.ReqIn, other)
	delete(b.ReqOut, me)
	if accept {
		now := time.Now()
		a.Friends[other], b.Friends[me] = now, now
	}
	return s.saveLocked(me, other)
}

func (s *SocialStore) Unfriend(me, other string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.getLocked(me).Friends, other)
	delete(s.getLocked(other).Friends, me)
	return s.saveLocked(me, other)
}

func (s *SocialStore) Block(me, other string, on bool) error {
	if me == other {
		return ErrSelf
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, b := s.getLocked(me), s.getLocked(other)
	if on {
		a.Blocks[other] = time.Now()
		delete(a.Friends, other)
		delete(b.Friends, me)
		delete(a.ReqIn, other)
		delete(a.ReqOut, other)
		delete(b.ReqIn, me)
		delete(b.ReqOut, me)
	} else {
		delete(a.Blocks, other)
	}
	return s.saveLocked(me, other)
}

// SendDM lưu tin ở cả hai phía trước khi giao, để mất mạng vẫn đồng bộ lại được (mục 11).
func (s *SocialStore) SendDM(from, to, text string) (DM, error) {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return DM{}, errors.New("Tin nhắn trống.")
	}
	if utf8.RuneCountInString(text) > DMMaxLen {
		text = string([]rune(text)[:DMMaxLen])
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	a, b := s.getLocked(from), s.getLocked(to)
	if _, x := a.Blocks[to]; x {
		return DM{}, ErrBlocked
	}
	if _, y := b.Blocks[from]; y {
		return DM{}, ErrBlocked
	}
	if _, ok := a.Friends[to]; !ok {
		return DM{}, errors.New("Chỉ nhắn riêng được với bạn bè.")
	}
	m := DM{ID: "dm" + randHex(8), From: from, Text: text, At: time.Now()}
	push := func(so *Social, other string) {
		l := append(so.DMs[other], m)
		if len(l) > DMKeep {
			l = l[len(l)-DMKeep:]
		}
		so.DMs[other] = l
	}
	push(a, to)
	push(b, from)
	b.Unread[from]++
	return m, s.saveLocked(from, to)
}

func (s *SocialStore) History(me, other string) []DM {
	s.mu.Lock()
	defer s.mu.Unlock()
	so := s.getLocked(me)
	if so.Unread[other] > 0 {
		delete(so.Unread, other)
		s.saveLocked(me)
	}
	return append([]DM{}, so.DMs[other]...)
}

type Report struct {
	ID       string    `json:"id"`
	Status   string    `json:"status"` // open | dismissed | actioned
	Handler  string    `json:"handler,omitempty"`
	Note     string    `json:"note,omitempty"`
	At       time.Time `json:"at"`
	Reporter string    `json:"reporter"`
	Target   string    `json:"target"`
	Reason   string    `json:"reason"`
	Name     string    `json:"name"`
	Room     string    `json:"room"`
	Evidence []ChatLog `json:"evidence"` // dòng chat lấy từ nhật ký máy chủ, không từ client
}

func (s *SocialStore) Report(r Report) error {
	if !contains(strings.Split(ReportReasons, "|"), r.Reason) {
		return errors.New("Lý do báo cáo không hợp lệ.")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Mỗi người tối đa ReportsPerDay báo cáo đang mở trong 24 giờ: chặn spam đẩy báo cáo thật ra khỏi hàng đợi.
	n := 0
	for _, x := range s.reports {
		if x.Reporter == r.Reporter && x.Status == "open" && r.At.Sub(x.At) < 24*time.Hour {
			n++
		}
	}
	if n >= ReportsPerDay {
		return errors.New("Bạn đã gửi nhiều báo cáo, kiểm duyệt viên đang xem. Thử lại sau nhé.")
	}
	r.ID, r.Status = "rp"+randHex(5), "open"
	s.reports = append(s.reports, r)
	s.reports = trimReports(s.reports, MaxReports)
	return writeAtomic(s.repPath, s.reports)
}

const (
	ReportsPerDay = 10
	MaxReports    = 2000
)

// trimReports giữ tối đa max báo cáo: bỏ báo cáo đã xử lý cũ nhất trước, chỉ bỏ báo cáo đang mở khi bất đắc dĩ.
func trimReports(rs []Report, max int) []Report {
	extra := len(rs) - max
	if extra <= 0 {
		return rs
	}
	out := rs[:0:0]
	for _, x := range rs {
		if extra > 0 && x.Status != "open" {
			extra--
			continue
		}
		out = append(out, x)
	}
	if len(out) > max {
		out = out[len(out)-max:]
	}
	return out
}

// Reports trả hàng đợi báo cáo cho kiểm duyệt viên, báo cáo đang mở lên trước.
func (s *SocialStore) Reports() []Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Report{}, s.reports...)
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Status == "open") != (out[j].Status == "open") {
			return out[i].Status == "open"
		}
		return out[i].At.After(out[j].At)
	})
	return out
}

func (s *SocialStore) ResolveReport(id, action, by, note string) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.reports {
		r := &s.reports[i]
		if r.ID != id {
			continue
		}
		if r.Status != "open" {
			return *r, errors.New("Báo cáo này đã được xử lý.")
		}
		r.Status, r.Handler, r.Note = "actioned", by, note
		if action == "dismiss" {
			r.Status = "dismissed"
		}
		return *r, writeAtomic(s.repPath, s.reports)
	}
	return Report{}, errors.New("Không tìm thấy báo cáo.")
}

func (s *SocialStore) Code(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id).Code
}

// AddMail gửi thư Chim Chíp; người bật "giảm thư" chỉ nhận thư công trình và hệ thống.
func (s *SocialStore) AddMail(id, kind, text string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	so := s.getLocked(id)
	if so.Privacy.QuietMail && kind != "build" && kind != "system" && kind != "hatch" {
		return false
	}
	so.Mail = append(so.Mail, MailItem{ID: "m" + randHex(5), Kind: kind, Text: text, At: time.Now()})
	if len(so.Mail) > MailKeep {
		so.Mail = so.Mail[len(so.Mail)-MailKeep:]
	}
	s.saveLocked(id)
	return true
}

func (s *SocialStore) ReadMail(id string) []MailItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	so := s.getLocked(id)
	out := append([]MailItem{}, so.Mail...) // luôn là mảng JSON, kể cả khi chưa có thư
	changed := false
	for i := range so.Mail {
		if !so.Mail[i].Read {
			so.Mail[i].Read, changed = true, true
		}
	}
	if changed {
		s.saveLocked(id)
	}
	return out
}

func (s *SocialStore) UnreadMail(id string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, m := range s.getLocked(id).Mail {
		if !m.Read {
			n++
		}
	}
	return n
}

// ---------- Bản nhìn ----------

type FriendView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Online bool   `json:"online"`
	Where  string `json:"where,omitempty"`
	Unread int    `json:"unread"`
}

type SocialView struct {
	Code    string       `json:"code"`
	Mail    int          `json:"mailUnread"`
	Friends []FriendView `json:"friends"`
	ReqIn   []FriendView `json:"reqIn"`
	ReqOut  []FriendView `json:"reqOut"`
	Blocks  []FriendView `json:"blocks"`
	Privacy Privacy      `json:"privacy"`
}

// View dựng danh sách; where(id) do hub cung cấp (đang online ở đâu, đã lọc theo quyền riêng tư).
func (s *SocialStore) View(id string, where func(string) (bool, string)) SocialView {
	s.mu.Lock()
	defer s.mu.Unlock()
	so := s.getLocked(id)
	mk := func(m map[string]time.Time, withPresence bool) []FriendView {
		out := []FriendView{}
		for oid := range m {
			o := s.getLocked(oid)
			fv := FriendView{ID: oid, Name: o.Name, Code: o.Code, Unread: so.Unread[oid]}
			if withPresence {
				on, w := where(oid)
				fv.Online = on
				if on && o.Privacy.ShowPresence {
					fv.Where = w
				}
			}
			out = append(out, fv)
		}
		return out
	}
	mail := 0
	for _, m := range so.Mail {
		if !m.Read {
			mail++
		}
	}
	return SocialView{Code: so.Code, Mail: mail, Friends: mk(so.Friends, true), ReqIn: mk(so.ReqIn, false), ReqOut: mk(so.ReqOut, false), Blocks: mk(so.Blocks, false), Privacy: so.Privacy}
}

func (s *SocialStore) Name(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getLocked(id).Name
}

func (s *SocialStore) FriendIDs(id string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for f := range s.getLocked(id).Friends {
		out = append(out, f)
	}
	return out
}
