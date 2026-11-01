package main

import (
	"strings"
	"testing"
	"time"
)

func (e *env) player(t *testing.T, user, name string) *client {
	tok, err := e.auth(t, "/api/register", user, "matkhau1")
	if err != "" {
		t.Fatal(err)
	}
	c := e.dial(t, user, tok)
	c.expect("need_char", 2*time.Second)
	c.send(msg{"t": "create_char", "name": name, "color": "vang", "comb": "don", "shirt": "none", "sex": "an", "tail": "xoan", "eyes": "sao"})
	c.expect("welcome", 2*time.Second)
	return c
}

// social lấy bản danh sách mới nhất: server cũng tự đẩy bản cập nhật nên bỏ các bản cũ còn trong hàng chờ.
func (c *client) social() map[string]any {
	keep := c.backlog[:0]
	for _, m := range c.backlog {
		if m["t"] != "social" {
			keep = append(keep, m)
		}
	}
	c.backlog = keep
	c.send(msg{"t": "social"})
	last := c.expect("social", 2*time.Second)
	for {
		select {
		case m := <-c.in:
			if m["t"] == "social" {
				last = m
			} else {
				c.backlog = append(c.backlog, m)
			}
		case <-time.After(150 * time.Millisecond):
			return last["social"].(map[string]any)
		}
	}
}

func (c *client) toastHas(sub string) {
	c.t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		m := c.expect("toast", time.Until(deadline))
		if strings.Contains(m["text"].(string), sub) {
			return
		}
	}
	c.t.Fatalf("%s: không có thông báo chứa %q", c.name, sub)
}

func TestSocialAndHomes(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a := e.player(t, "alice", "Gà Bông")
	b := e.player(t, "bob", "Gà Mơ")
	c := e.player(t, "carol", "Gà Tím")

	// --- Kết bạn bằng mã người chơi ---
	codeA := a.social()["code"].(string)
	if len(codeA) != 6 {
		t.Fatalf("mã người chơi: %q", codeA)
	}
	b.send(msg{"t": "friend_add", "id": codeA})
	b.toastHas("Đã gửi lời mời")
	a.toastHas("muốn kết bạn")
	reqIn := a.social()["reqIn"].([]any)
	bobID := reqIn[0].(map[string]any)["id"].(string)
	a.send(msg{"t": "friend_respond", "id": bobID, "on": true})
	b.toastHas("đã nhận lời")
	fr := a.social()["friends"].([]any)
	if len(fr) != 1 || fr[0].(map[string]any)["online"] != true {
		t.Fatalf("bạn bè + trạng thái online: %v", fr)
	}
	aliceID := b.social()["friends"].([]any)[0].(map[string]any)["id"].(string)

	// --- Tin riêng chỉ giữa bạn bè, lưu lại để đồng bộ ---
	b.send(msg{"t": "dm", "id": aliceID, "text": "  ê đi đào giun không?  "})
	dm := a.expect("dm", 2*time.Second)
	if dm["msg"].(map[string]any)["text"] != "ê đi đào giun không?" {
		t.Fatalf("dm: %v", dm)
	}
	a.send(msg{"t": "dm_history", "id": bobID})
	if h := a.expect("dm_history", 2*time.Second)["msgs"].([]any); len(h) != 1 {
		t.Fatal("lịch sử tin riêng")
	}
	c.send(msg{"t": "dm", "id": aliceID, "text": "hello"})
	c.toastHas("bạn bè")

	// --- Nhà riêng: vào nhà mình, đặt đồ đúng luật ---
	for _, x := range []*client{a, b, c} {
		e.warp(x, "neighborhood") // nhà riêng mở từ cổng ở Xóm Rơm Vàng
	}
	a.send(msg{"t": "visit", "owner": "me"})
	wh := a.expect("welcome", 2*time.Second)
	if wh["zone"] != "home" || wh["home"] == nil || wh["room"].(map[string]any)["kind"] != "home" {
		t.Fatalf("vào chuồng riêng: %v", wh["room"])
	}
	if r := a.act("home_place", msg{"id": "chau_hoa", "x": 0, "z": 3}); r["ok"] == true {
		t.Fatal("không được đặt chắn lối vào")
	}
	r := a.mustOK("home_place", msg{"id": "chau_hoa", "x": -2, "z": -2, "i": 1})
	if r["data"].(map[string]any)["obj"].(map[string]any)["rot"].(float64) != 1 {
		t.Fatal("xoay 90°")
	}
	if r := a.act("home_build", msg{"id": "do"}); r["ok"] == true {
		t.Fatal("phải đứng cạnh Bò Mộc / đủ hạng đất")
	}

	// --- Bạn bè vào được (quyền mặc định "bạn bè"), người lạ thì không ---
	b.send(msg{"t": "visit", "owner": aliceID})
	wb := b.expect("welcome", 2*time.Second)
	if wb["zone"] != "home" || len(wb["players"].([]any)) != 1 {
		t.Fatal("bob thấy alice trong chuồng riêng")
	}
	if r := b.act("home_place", msg{"id": "chau_hoa", "x": 2, "z": 0}); r["ok"] == true {
		t.Fatal("khách không được sửa nhà")
	}
	c.send(msg{"t": "visit", "owner": aliceID})
	c.toastHas("đóng cửa")

	// --- Công khai thì người lạ vào được; chủ nhà mời khách ra ---
	a.mustOK("home_access", msg{"id": "public"})
	c.send(msg{"t": "visit", "owner": aliceID})
	c.expect("welcome", 2*time.Second)
	var carolPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Tím" {
			carolPID = p["pid"].(string)
			break
		}
	}
	a.mustOK("home_kick", msg{"target": carolPID})
	if back := c.expect("welcome", 2*time.Second); back["zone"] != "neighborhood" || back["room"].(map[string]any)["kind"] != "public" {
		t.Fatal("khách bị mời ra phải đứng ở cổng Xóm Rơm Vàng")
	}

	// --- Chặn: không vào nhà, không kết bạn, không nhận chat ---
	a.send(msg{"t": "block", "id": c.socialCodeOf(t), "on": true})
	a.toastHas("Đã chặn")
	c.send(msg{"t": "visit", "owner": aliceID})
	c.toastHas("Không thể")
	c.send(msg{"t": "friend_add", "id": codeA})
	c.toastHas("Không thể")

	// --- Rời nhà về sảnh; nhà trống thì phòng đóng ---
	b.send(msg{"t": "leave_home"})
	if wl := b.expect("welcome", 2*time.Second); wl["zone"] != "neighborhood" {
		t.Fatal("rời nhà phải ra cổng Xóm Rơm Vàng")
	}
	a.mustOK("go", msg{"to": "lobby"})
	a.expect("welcome", 2*time.Second)
	time.Sleep(100 * time.Millisecond)
	e.hub.mu.Lock()
	open := len(e.hub.homes)
	e.hub.mu.Unlock()
	if open != 0 {
		t.Fatal("chuồng riêng trống phải được đóng")
	}

	// --- Báo cáo lưu kèm nhật ký chat của máy chủ ---
	b.send(msg{"t": "chat", "text": "nội dung xấu"})
	var bPID string
	for {
		m := a.expect("chat", 2*time.Second)
		if m["kind"] == "chat" {
			bPID = m["pid"].(string)
			break
		}
	}
	a.send(msg{"t": "report", "target": bPID, "reason": "ngon_tu"})
	a.toastHas("Đã gửi báo cáo")
}

// socialCodeOf lấy mã người chơi của chính client này.
func (c *client) socialCodeOf(t *testing.T) string {
	return c.social()["code"].(string)
}
