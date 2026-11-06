package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Store lưu tài khoản thành file JSON (ghi tạm rồi rename để không hỏng file khi lỗi giữa chừng)
// và sổ cái thóc dạng append-only. Bản production thay bằng PostgreSQL theo mục 16 của plan;
// giao diện Store giữ nguyên nên phần game không phải sửa.

type userRec struct {
	ID      string `json:"id"`
	Salt    string `json:"salt"`
	Hash    string `json:"hash"`
	RecSalt string `json:"recSalt"` // mã khôi phục chỉ lưu dạng băm
	RecHash string `json:"recHash"`
}

type Store struct {
	dir string

	mu       sync.Mutex
	users    map[string]userRec  // username (chữ thường) → bản ghi đăng nhập
	accounts map[string]*Account // cache đã nạp
	tokens   map[string]string   // token → account ID
	ledgerMu sync.Mutex
	ledgerF  *os.File

	Social    *SocialStore
	Homes     *HomeStore
	Mod       *Moderation
	Contracts *ContractStore
	Guilds    *GuildStore

	attempts map[string][]time.Time // username → lần thử sai gần đây
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "accounts"), 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, users: map[string]userRec{}, accounts: map[string]*Account{}, tokens: map[string]string{}}
	if b, err := os.ReadFile(filepath.Join(dir, "users.json")); err == nil {
		if err := json.Unmarshal(b, &s.users); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(filepath.Join(dir, "ledger.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	s.ledgerF = f
	if s.Social, err = OpenSocial(dir); err != nil {
		return nil, err
	}
	if s.Homes, err = OpenHomes(dir); err != nil {
		return nil, err
	}
	if s.Mod, err = OpenModeration(dir); err != nil {
		return nil, err
	}
	if s.Contracts, err = OpenContracts(dir); err != nil {
		return nil, err
	}
	if s.Guilds, err = OpenGuilds(dir); err != nil {
		return nil, err
	}
	s.attempts = map[string][]time.Time{}
	return s, nil
}

func writeAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	// Không fsync: mỗi giao dịch đều ghi trên goroutine phòng, fsync (5–30 ms trên Windows) làm giật tick 20 Hz.
	// Ghi file tạm rồi rename vẫn nguyên tử khi tiến trình chết; mất điện cả máy là rủi ro chấp nhận cho tới khi lên PostgreSQL.
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return renameWithRetry(tmp, path)
}

// renameWithRetry đổi tên, thử lại khi Windows từ chối tạm thời (file đích đang bị trình diệt virus/đánh chỉ mục mở).
// Chờ 1, 2, 4… 64 ms: tối đa ~130 ms, chỉ khi gặp lỗi; lỗi khác trả ngay. Nhiều kho lưu lúc đang giữ khoá chung
// (hợp đồng, hội, Store) nên lúc thử lại phòng khác có thể chờ theo; log để đo xem thực tế tốn bao lâu.
func renameWithRetry(from, to string) error {
	err := os.Rename(from, to)
	waited := time.Duration(0)
	for wait := time.Millisecond; err != nil && renameRetryable(err) && wait <= 64*time.Millisecond; wait *= 2 {
		time.Sleep(wait)
		waited += wait
		err = os.Rename(from, to)
	}
	switch {
	case waited > 0 && err == nil:
		log.Printf("lưu %s: đổi tên được sau khi thử lại %v (file đang bị tiến trình khác mở)", to, waited)
	case waited > 0:
		log.Printf("lưu %s: thử lại %v vẫn không đổi tên được: %v", to, waited, err)
	}
	return err
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AccountIDLen: account ID là randHex(12) = 24 ký tự hex thường.
const AccountIDLen = 24

// validAccountID chặn mọi ID không đúng dạng do server sinh ra. Account ID được ghép thẳng vào đường dẫn file
// (accounts/, social/, homes/), nên ID từ client như "../accounts/<id>" phải bị loại trước khi chạm đĩa.
func validAccountID(id string) bool {
	if len(id) != AccountIDLen {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

var usernameRe = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)

var (
	ErrUserTaken   = errors.New("Tên đăng nhập đã có người dùng.")
	ErrBadLogin    = errors.New("Sai tên đăng nhập hoặc mật khẩu.")
	ErrBadUsername = errors.New("Tên đăng nhập 3–20 ký tự: chữ thường không dấu, số hoặc _.")
	ErrBadPassword = errors.New("Mật khẩu cần ít nhất 6 ký tự.")
	ErrTooMany     = errors.New("Thử sai quá nhiều lần. Chờ 10 phút rồi thử lại.")
	ErrBadRecovery = errors.New("Sai tên đăng nhập hoặc mã khôi phục.")
)

const maxAttempts = 8

// throttled kiểm tra và ghi nhận lần thử sai theo tên đăng nhập (chống dò mật khẩu).
func (s *Store) throttled(username string, failed bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	cut := time.Now().Add(-10 * time.Minute)
	l := s.attempts[username][:0]
	for _, t := range s.attempts[username] {
		if t.After(cut) {
			l = append(l, t)
		}
	}
	if failed {
		l = append(l, time.Now())
	}
	s.attempts[username] = l
	return len(l) >= maxAttempts
}

// Mã khôi phục dạng XXXX-XXXX-XXXX dễ chép tay.
func newRecoveryCode() string {
	b := []byte(randHex(12))
	out := make([]byte, 0, 14)
	for i := 0; i < 12; i++ {
		if i > 0 && i%4 == 0 {
			out = append(out, '-')
		}
		out = append(out, codeAlphabet[int(b[i])%len(codeAlphabet)])
	}
	return string(out)
}

func normRecovery(c string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(c), " ", ""), "-", ""))
}

func hashPassword(pw, salt string) string {
	k, _ := pbkdf2.Key(sha256.New, pw, []byte(salt), 120_000, 32)
	return hex.EncodeToString(k)
}

func (s *Store) Register(username, password string) (string, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if !usernameRe.MatchString(username) {
		return "", "", ErrBadUsername
	}
	if len(password) < 6 {
		return "", "", ErrBadPassword
	}
	salt := randHex(16)
	hash := hashPassword(password, salt)
	rec := newRecoveryCode()
	recSalt := randHex(16)
	recHash := hashPassword(normRecovery(rec), recSalt)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[username]; ok {
		return "", "", ErrUserTaken
	}
	id := randHex(12)
	acc := NewAccount(id, username, time.Now())
	if err := writeAtomic(s.accountPath(id), acc); err != nil {
		return "", "", err
	}
	s.users[username] = userRec{ID: id, Salt: salt, Hash: hash, RecSalt: recSalt, RecHash: recHash}
	if err := writeAtomic(filepath.Join(s.dir, "users.json"), s.users); err != nil {
		delete(s.users, username)
		return "", "", err
	}
	s.accounts[id] = acc
	return s.issueTokenLocked(id), rec, nil
}

// Recover đặt lại mật khẩu bằng mã khôi phục; cấp mã mới và huỷ mọi phiên đăng nhập cũ (mục 14).
func (s *Store) Recover(username, code, newPassword string) (string, string, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if s.throttled(username, false) {
		return "", "", "", ErrTooMany
	}
	if len(newPassword) < 6 {
		return "", "", "", ErrBadPassword
	}
	s.mu.Lock()
	u, ok := s.users[username]
	s.mu.Unlock()
	if !ok || u.RecHash == "" || subtle.ConstantTimeCompare([]byte(hashPassword(normRecovery(code), u.RecSalt)), []byte(u.RecHash)) != 1 {
		s.throttled(username, true)
		return "", "", "", ErrBadRecovery
	}
	rec := newRecoveryCode()
	u.Salt = randHex(16)
	u.Hash = hashPassword(newPassword, u.Salt)
	u.RecSalt = randHex(16)
	u.RecHash = hashPassword(normRecovery(rec), u.RecSalt)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[username] = u
	if err := writeAtomic(filepath.Join(s.dir, "users.json"), s.users); err != nil {
		return "", "", "", err
	}
	for t, id := range s.tokens {
		if id == u.ID {
			delete(s.tokens, t)
		}
	}
	return s.issueTokenLocked(u.ID), rec, u.ID, nil
}

func (s *Store) Login(username, password string) (string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if s.throttled(username, false) {
		return "", ErrTooMany
	}
	s.mu.Lock()
	u, ok := s.users[username]
	s.mu.Unlock()
	if !ok {
		hashPassword(password, "timing-equalizer")
		s.throttled(username, true)
		return "", ErrBadLogin
	}
	if subtle.ConstantTimeCompare([]byte(hashPassword(password, u.Salt)), []byte(u.Hash)) != 1 {
		s.throttled(username, true)
		return "", ErrBadLogin
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.issueTokenLocked(u.ID), nil
}

func (s *Store) issueTokenLocked(id string) string {
	t := randHex(24)
	s.tokens[t] = id
	return t
}

func (s *Store) AccountIDForToken(t string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.tokens[t]
	return id, ok
}

func (s *Store) accountPath(id string) string {
	return filepath.Join(s.dir, "accounts", id+".json")
}

// Account trả về con trỏ dùng chung. Chỉ phiên đang điều khiển (một phiên/tài khoản) được sửa nó.
func (s *Store) Account(id string) (*Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.accounts[id]; ok {
		return a, nil
	}
	if !validAccountID(id) {
		return nil, ErrNotFound
	}
	b, err := os.ReadFile(s.accountPath(id))
	if err != nil {
		return nil, err
	}
	var a Account
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	a.normalize()
	s.accounts[id] = &a
	return &a, nil
}

// Save ghi bản lưu bền vững và sổ cái. Trả lỗi để caller không báo "thành công giả" (mục 17).
// Close đóng các file append-only (sổ cái, nhật ký quản trị). Ghi sau khi đóng chỉ ra lỗi log, không panic.
func (s *Store) Close() error {
	s.ledgerMu.Lock()
	err := s.ledgerF.Close()
	s.ledgerMu.Unlock()
	if e := s.Mod.Close(); err == nil {
		err = e
	}
	return err
}

func (s *Store) Save(a *Account) error {
	if err := writeAtomic(s.accountPath(a.ID), a); err != nil {
		return err
	}
	if l := a.TakeLedger(); len(l) > 0 {
		s.ledgerMu.Lock()
		defer s.ledgerMu.Unlock()
		// Tài khoản đã lưu là nguồn đúng; lỗi ghi sổ cái chỉ báo log để đối soát, không hoàn nguyên giao dịch.
		enc := json.NewEncoder(s.ledgerF)
		for _, e := range l {
			if err := enc.Encode(e); err != nil {
				log.Printf("ledger %s: %v", a.ID, err)
			}
		}
	}
	return nil
}
