package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Các phần hoàn thiện V1: nối lại phiên, khôi phục tài khoản, quản trị/kiểm duyệt, cộng đồng.

func (e *env) registerFull(t *testing.T, user string) (token, recovery string) {
	b, _ := json.Marshal(map[string]string{"username": user, "password": "matkhau1"})
	r, err := http.Post(e.srv.URL+"/api/register", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out map[string]string
	json.NewDecoder(r.Body).Decode(&out)
	if out["err"] != "" {
		t.Fatal(out["err"])
	}
	return out["token"], out["recovery"]
}

func (e *env) adminCall(t *testing.T, token, method, path string, body any) (int, map[string]any) {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rd)
	req.Header.Set("Authorization", "Bearer "+token)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	var out map[string]any
	json.NewDecoder(r.Body).Decode(&out)
	return r.StatusCode, out
}

func (e *env) playerTok(t *testing.T, user, name string) (*client, string) {
	tok, _ := e.registerFull(t, user)
	c := e.dial(t, user, tok)
	c.expect("need_char", 2*time.Second)
	c.send(msg{"t": "create_char", "name": name, "color": "vang", "comb": "don", "shirt": "none", "sex": "an"})
	c.expect("welcome", 2*time.Second)
	return c, tok
}

func TestReconnectDuringPK(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.ReconnectGrace = 2 * time.Second })
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	b, tokB := e.playerTok(t, "bob", "Gà Mơ")
	e.warp(a, "arena")
	e.warp(b, "arena")
	var bPID string
	for {
		m := a.expect("pjoin", 2*time.Second)
		if p := m["p"].(map[string]any); p["name"] == "Gà Mơ" {
			bPID = p["pid"].(string)
			break
		}
	}
	a.mustOK("pk_invite", msg{"target": bPID, "mode": "coin"})
	inv := b.expect("pk_invite", 2*time.Second)
	b.mustOK("pk_accept", msg{"id": inv["id"]})
	st := a.expect("pk_start", 2*time.Second)
	b.expect("pk_start", 2*time.Second)

	// Bob rớt mạng giữa trận rồi tải lại trang (resume=false): vẫn phải về đúng trận, không xử thua.
	b.conn.Close()
	if g := a.expect("pgone", 2*time.Second); g["gone"] != true {
		t.Fatal("người khác thấy bob mất kết nối")
	}
	time.Sleep(300 * time.Millisecond)
	b2 := e.dial(t, "bob2", tokB)
	w := b2.expect("welcome", 2*time.Second)
	if w["zone"] != "arena" {
		t.Fatalf("nối lại trong trận phải về sân tỷ thí, đang ở %v", w["zone"])
	}
	res := b2.expect("pk_resume", 2*time.Second)
	if res["id"] != st["id"] {
		t.Fatal("phải khôi phục đúng trận")
	}
	if g := a.expect("pgone", 2*time.Second); g["gone"] != false {
		t.Fatal("người khác thấy bob đã quay lại")
	}
	// Đấu hết 3 lượt: ai tới lượt thì chọn.
	players := map[float64]*client{st["you"].(float64): a, 1 - st["you"].(float64): b2}
	for round := 0; round < CoinRounds; round++ {
		r := a.expect("pk_round", 5*time.Second)
		players[r["chooser"].(float64)].mustOK("pk_pick", msg{"id": st["id"], "side": "ngua"})
	}
	end := b2.expect("pk_end", 6*time.Second)
	if end["reason"] != "" || end["winner"].(float64) < 0 {
		t.Fatalf("trận phải kết thúc bình thường: %v", end)
	}
}

func TestReconnectGraceAndResume(t *testing.T) {
	e := newEnvBal(t, t.TempDir(), func(b *Balance) { b.ReconnectGrace = time.Second })
	a, _ := e.playerTok(t, "alice", "Gà Bông")
	c, tokC := e.playerTok(t, "carol", "Gà Tím")
	a.expect("pjoin", 2*time.Second)

	// Tự nối lại (resume) ngoài trận: về đúng khu và vị trí cũ.
	e.warp(c, "market")
	posBefore := c.position()
	c.conn.Close()
	time.Sleep(200 * time.Millisecond)
	c3 := e.dialResume(t, "carol3", tokC, true)
	w := c3.expect("welcome", 2*time.Second)
	if w["zone"] != "market" {
		t.Fatalf("nối lại phải giữ khu: %v", w["zone"])
	}
	if p := w["pos"].(map[string]any); (Vec{p["x"].(float64), p["z"].(float64)}).Dist(posBefore) > 0.5 {
		t.Fatal("nối lại phải giữ vị trí")
	}

	// Mất kết nối quá thời gian giữ chỗ: rời phòng; lần vào sau là phiên mới ở sảnh.
	c3.conn.Close()
	time.Sleep(e.hub.bal.ReconnectGrace + 600*time.Millisecond)
	c4 := e.dialResume(t, "carol4", tokC, true)
	if w := c4.expect("welcome", 2*time.Second); w["zone"] != ZoneLobby {
		t.Fatalf("hết giờ giữ chỗ phải vào sảnh: %v", w["zone"])
	}
}

func (e *env) dialResume(t *testing.T, name, token string, resume bool) *client {
	c := e.dial(t, name, "")
	c.send(msg{"t": "hello", "token": token, "resume": resume})
	return c
}

func TestRecoveryAndThrottle(t *testing.T) {
	e := newEnv(t, t.TempDir())
	tok, rec := e.registerFull(t, "dung")
	if len(rec) != 14 {
		t.Fatalf("mã khôi phục: %q", rec)
	}
	post := func(path string, body map[string]string) map[string]string {
		b, _ := json.Marshal(body)
		r, err := http.Post(e.srv.URL+path, "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()
		var out map[string]string
		json.NewDecoder(r.Body).Decode(&out)
		return out
	}
	if out := post("/api/recover", map[string]string{"username": "dung", "code": "AAAA-BBBB-CCCC", "password": "moi12345"}); out["err"] == "" {
		t.Fatal("mã sai phải bị từ chối")
	}
	out := post("/api/recover", map[string]string{"username": "dung", "code": strings.ToLower(rec), "password": "moi12345"})
	if out["err"] != "" || out["recovery"] == rec || out["token"] == "" {
		t.Fatalf("khôi phục: %v", out)
	}
	// Token cũ hết hiệu lực, mật khẩu mới dùng được, mã cũ không dùng lại được.
	old := e.dial(t, "old", tok)
	old.expect("auth_fail", 2*time.Second)
	if _, err := e.auth(t, "/api/login", "dung", "moi12345"); err != "" {
		t.Fatal(err)
	}
	if o := post("/api/recover", map[string]string{"username": "dung", "code": rec, "password": "khac1234"}); o["err"] == "" {
		t.Fatal("mã khôi phục cũ phải hết hiệu lực")
	}
	// Đăng nhập sai liên tục bị tạm chặn, kể cả lần nhập đúng sau đó.
	for i := 0; i < maxAttempts; i++ {
		e.auth(t, "/api/login", "dung", "sai")
	}
	if _, err := e.auth(t, "/api/login", "dung", "moi12345"); !strings.Contains(err, "quá nhiều") {
		t.Fatalf("phải tạm chặn sau nhiều lần sai: %q", err)
	}
}

func TestAdminModeration(t *testing.T) {
	e := newEnv(t, t.TempDir())
	e.hub.staff = ParseStaff("modx:mod,econx:econ,boss:admin")
	modTok, _ := e.registerFull(t, "modx")
	econTok, _ := e.registerFull(t, "econx")
	bossTok, _ := e.registerFull(t, "boss")
	p, _ := e.playerTok(t, "phong", "Gà Phong")
	q, tokQ := e.playerTok(t, "quan", "Gà Quân")
	code := q.social()["code"].(string)

	if c, _ := e.adminCall(t, tokQ, "GET", "/api/admin/me", nil); c != 403 {
		t.Fatal("người chơi thường không vào được trang quản trị")
	}
	if c, out := e.adminCall(t, modTok, "GET", "/api/admin/overview", nil); c != 200 || out["online"].(float64) != 2 {
		t.Fatalf("tổng quan: %d %v", c, out)
	}
	// Quyền tách: econ không kiểm duyệt, mod không xem sổ cái.
	if c, _ := e.adminCall(t, econTok, "POST", "/api/admin/moderate", map[string]any{"code": code, "action": "mute", "minutes": 5, "reason": "spam"}); c != 403 {
		t.Fatal("econ không được kiểm duyệt")
	}
	if c, _ := e.adminCall(t, modTok, "GET", "/api/admin/ledger?code="+code, nil); c != 403 {
		t.Fatal("mod không được xem sổ cái")
	}
	if c, out := e.adminCall(t, econTok, "GET", "/api/admin/ledger?code="+code, nil); c != 200 || out["name"] != "Gà Quân" {
		t.Fatalf("econ xem sổ cái: %d %v", c, out)
	}

	// Báo cáo → hàng đợi → xử lý bằng mute: người bị mute không chat được.
	q.send(msg{"t": "chat", "text": "spam spam"})
	var qPID string
	for {
		m := p.expect("chat", 2*time.Second)
		if m["kind"] == "chat" {
			qPID = m["pid"].(string)
			break
		}
	}
	p.send(msg{"t": "report", "target": qPID, "reason": "spam"})
	p.toastHas("Đã gửi báo cáo")
	_, reps := e.adminCall(t, modTok, "GET", "/api/admin/reports", nil)
	rep := reps["reports"].([]any)[0].(map[string]any)
	if rep["status"] != "open" || len(rep["evidence"].([]any)) == 0 {
		t.Fatalf("báo cáo phải mở và có bằng chứng: %v", rep)
	}
	if c, out := e.adminCall(t, modTok, "POST", "/api/admin/reports/resolve", map[string]any{"id": rep["id"], "action": "mute", "minutes": 30, "note": "nhắc nhở"}); c != 200 {
		t.Fatalf("xử lý báo cáo: %v", out)
	}
	q.toastHas("khoá chat")
	q.send(msg{"t": "chat", "text": "còn nói được không"})
	q.toastHas("khoá chat")
	if c, _ := e.adminCall(t, modTok, "POST", "/api/admin/reports/resolve", map[string]any{"id": rep["id"], "action": "dismiss"}); c == 200 {
		t.Fatal("báo cáo đã xử lý không xử lý lại")
	}

	// Ban: bị kick ngay và không vào lại được.
	if c, _ := e.adminCall(t, modTok, "POST", "/api/admin/moderate", map[string]any{"code": code, "action": "ban", "minutes": 60, "reason": "gian lận"}); c != 200 {
		t.Fatal("ban")
	}
	q.expect("kicked", 2*time.Second)
	q2 := e.dial(t, "quan2", tokQ)
	if f := q2.expect("auth_fail", 2*time.Second); !strings.Contains(f["err"].(string), "khoá") {
		t.Fatal("tài khoản bị ban không vào được")
	}

	// Bảo trì: chặn người mới vào, nhân sự vẫn vào được; mọi thao tác có nhật ký.
	if c, _ := e.adminCall(t, modTok, "POST", "/api/admin/maintenance", map[string]any{"on": true}); c != 403 {
		t.Fatal("chỉ admin bật bảo trì")
	}
	e.adminCall(t, bossTok, "POST", "/api/admin/maintenance", map[string]any{"on": true, "text": "nâng cấp"})
	_, rTok := e.registerFull(t, "rau")
	_ = rTok
	tokR, _ := e.auth(t, "/api/login", "rau", "matkhau1")
	if m := e.dial(t, "rau", tokR).expect("maintenance", 2*time.Second); m["text"] != "nâng cấp" {
		t.Fatal("bảo trì phải chặn người mới vào")
	}
	_, audit := e.adminCall(t, bossTok, "GET", "/api/admin/audit", nil)
	if len(audit["lines"].([]any)) < 4 {
		t.Fatalf("nhật ký quản trị: %v", audit)
	}
}

func TestCommunityRules(t *testing.T) {
	a := newAcc()
	a.Worms["dat"] = BagCapacity
	if err := a.CanDig(t0, PlanBalance()); err == nil || !strings.Contains(err.Error(), "Túi đầy") {
		t.Fatal("túi đầy phải dừng lượt đào trước khi bắt đầu")
	}
	orders := OrdersFor(DayKey(t0))
	if len(orders) != 3 || orders[0].Worm != "dat" || orders[1].ID == orders[2].ID {
		t.Fatalf("3 đơn/ngày, có đơn giun đất cho người mới: %+v", orders)
	}
	a.Worms["dat"] = 10
	thoc := a.Thoc
	if _, _, err := a.DeliverOrder(orders[0].ID, t0); err != nil || a.Thoc != thoc+orders[0].Reward {
		t.Fatalf("giao đơn: %v", err)
	}
	if _, _, err := a.DeliverOrder(orders[0].ID, t0); err == nil {
		t.Fatal("mỗi đơn một lần/ngày")
	}
	if notes := a.Discovered("zone:lobby"); len(notes) != 0 || a.Discovered("zone:lobby") != nil {
		t.Fatal("ghi khám phá một lần")
	}
	for _, z := range discoverZones {
		a.Discovered("zone:" + z)
	}
	if !a.hasTitle("tham_hiem") || a.SetTitle("tham_hiem") != nil || a.SetTitle("cao_thu") == nil {
		t.Fatal("mở và đeo danh hiệu")
	}

	f, _ := OpenFestival(t.TempDir())
	if _, err := f.Add(f.Snapshot().Goal + 1); err == nil {
		t.Fatal("góp vượt trần phải bị từ chối")
	}
	st, err := f.Add(f.Snapshot().Goal)
	if err != nil || !st.Unlocked {
		t.Fatal("đủ mục tiêu thì mở trang trí")
	}
	if _, err := f.Add(10); err == nil {
		t.Fatal("đã đủ thì không nhận thêm")
	}
	f.Undo(10)
	if f.Snapshot().Unlocked {
		t.Fatal("hoàn phần góp phải cập nhật trạng thái")
	}
}

func TestMailOnVisitAndFestivalE2E(t *testing.T) {
	e := newEnv(t, t.TempDir())
	a := e.player(t, "alice", "Gà Bông")
	b := e.player(t, "bob", "Gà Mơ")
	codeA := a.social()["code"].(string)
	b.send(msg{"t": "friend_add", "id": codeA})
	a.expect("mail_new", 2*time.Second)
	bobID := a.social()["reqIn"].([]any)[0].(map[string]any)["id"].(string)
	a.send(msg{"t": "friend_respond", "id": bobID, "on": true})
	aliceID := b.social()["friends"].([]any)[0].(map[string]any)["id"].(string)
	e.warp(b, "neighborhood")
	b.send(msg{"t": "visit", "owner": aliceID})
	b.expect("welcome", 2*time.Second)
	a.expect("mail_new", 2*time.Second)
	a.send(msg{"t": "mail"})
	mail := a.expect("mail", 2*time.Second)["mail"].([]any)
	if len(mail) != 2 || !strings.Contains(mail[1].(map[string]any)["text"].(string), "ghé thăm") {
		t.Fatalf("thư Chim Chíp: %v", mail)
	}
	// Góp ngày hội cần đứng ở bảng tin và đủ thóc.
	if r := a.act("festival_give", msg{"i": 10}); r["ok"] == true {
		t.Fatal("phải đứng ở bảng tin")
	}
}
