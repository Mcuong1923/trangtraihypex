package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test hồi quy cho các lỗi nghiêm trọng tìm thấy khi rà logic 05/11/2026
// (plans/reports/code-reviewer-261105-1412-game-logic-review.md, mục C1–C3; C4 nằm trong TestContractStoreRules).

// C1: account ID từ client được ghép vào đường dẫn file; "../accounts/<id>" từng ghi đè được tài khoản người khác.
func TestAccountIDPathTraversalBlocked(t *testing.T) {
	good := randHex(12)
	if !validAccountID(good) {
		t.Fatalf("ID do server sinh phải hợp lệ: %q", good)
	}
	for _, bad := range []string{"", "../users", "../accounts/" + good, strings.ToUpper(good), good[:23] + "/", good + "0", "..%2f..%2f..%2fusers1234"} {
		if validAccountID(bad) {
			t.Fatalf("ID %q phải bị từ chối", bad)
		}
	}

	dir := t.TempDir()
	st, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	victim := randHex(12)
	victimPath := filepath.Join(dir, "accounts", victim+".json")
	orig := []byte(`{"id":"` + victim + `","thoc":999}`)
	if err := os.WriteFile(victimPath, orig, 0o644); err != nil {
		t.Fatal(err)
	}
	so, err := OpenSocial(dir)
	if err != nil {
		t.Fatal(err)
	}
	me := randHex(12)
	for _, bad := range []string{"../accounts/" + victim, "../users"} {
		if err := so.Unfriend(me, bad); err == nil {
			t.Fatalf("Unfriend với ID %q phải lỗi", bad)
		}
		if err := so.Block(me, bad, true); err == nil {
			t.Fatalf("Block với ID %q phải lỗi", bad)
		}
	}
	if b, _ := os.ReadFile(victimPath); string(b) != string(orig) {
		t.Fatalf("file tài khoản nạn nhân bị ghi đè: %s", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "users.json")); err == nil {
		t.Fatal("không được tạo/ghi đè users.json qua store xã hội")
	}
	if _, err := st.Account("../accounts/" + victim); err == nil { // code cũ đọc được file nạn nhân qua đường dẫn này
		t.Fatal("Store.Account phải từ chối ID lạ")
	}
	hs, err := OpenHomes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hs.Get("../accounts/"+victim, "x"); err == nil {
		t.Fatal("HomeStore.Get phải từ chối ID lạ")
	}
	if b, _ := os.ReadFile(victimPath); string(b) != string(orig) {
		t.Fatalf("file tài khoản nạn nhân bị ghi đè qua nhà: %s", b)
	}
}

// C2: bán giun với ID lạ từng deref nil và làm sập cả tiến trình.
func TestSellUnknownWormDoesNotCrash(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a, _ := e.seeded(t, "banla", "Gà Bán Lạ", func(*Account) {})
	e.warp(a, "market")
	a.walkToItem("market", "sap_giun")
	for i := 0; i < 2; i++ {
		if r := a.act("sell", msg{"id": "khong_co_giun_nay"}); r["ok"] == true {
			t.Fatal("bán giun không tồn tại phải bị từ chối")
		}
	}
	// Phòng vẫn sống: hành động sau đó vẫn có phản hồi.
	if r := a.act("sell", msg{"id": Worms[0].ID}); r["ok"] == true {
		t.Fatal("túi rỗng thì không bán được")
	}
}

// C3: đổi kênh giữa lượt đào từng xoá phiên đào mà không tính hồi đất → đào lại ngay để gieo lại loài giun.
func TestSwitchRoomKeepsDigCooldown(t *testing.T) {
	// Hồi đất 5 giây (test mặc định 300 ms) để lệnh đào lại sau khi đổi kênh chắc chắn rơi vào thời gian hồi.
	// Đặt cả DigCooldownSec vì tune chạy sau finalize.
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.DigCooldown, b.DigCooldownSec = 5*time.Second, 5 })
	a, _ := e.seeded(t, "doikenh", "Gà Đổi Kênh", func(acc *Account) { acc.Mana = 100 })
	e.warp(a, "garden")
	a.walkToItem("garden", "dat_a")
	a.mustOK("dig_start", msg{"id": "dat_a"})
	a.send(msg{"t": "switch_room", "room": "new"})
	a.expect("welcome", 2*time.Second)
	a.mu.Lock()
	zone := a.zone
	a.mu.Unlock()
	if zone != "garden" {
		t.Fatalf("đổi kênh phải giữ chỗ đứng ở vườn, đang ở %s", zone)
	}
	if a.num("digReadyIn") <= 0 {
		t.Fatal("rời phòng giữa lượt đào phải tính hồi đất")
	}
	if r := a.act("dig_start", msg{"id": "dat_a"}); r["ok"] == true {
		t.Fatal("đổi kênh rồi đào lại ngay phải bị chặn bởi hồi đất")
	}
}
