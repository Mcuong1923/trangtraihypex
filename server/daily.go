package main

import (
	"errors"
	"time"
)

// Điểm danh 7 ngày (PLAN_LOGIC_GAME mục 16): vào game mỗi ngày nhận quà tăng dần, ngày 7 là trứng đốm.
// Lỡ một ngày thì chuỗi quay về ngày 1. Cụ Cục trao quà bằng bong bóng khi vào game.

type CheckIn struct {
	Day    string `json:"day"`    // ngày (giờ Việt Nam) đã điểm danh gần nhất
	Streak int    `json:"streak"` // 1..7
}

type CheckInView struct {
	Today  bool `json:"today"`  // hôm nay đã điểm danh
	Streak int  `json:"streak"` // chuỗi hiện tại
	Next   int  `json:"next"`   // ngày thứ mấy nếu điểm danh hôm nay / ngày mai
}

var CheckInRewards = []Reward{
	// PLAN_LOGIC_GAME 16: ngày 1–6 là 20/30/40/50/60/80 thóc kèm cám hoặc hạt giống; ngày 7 một trứng đốm.
	{Thoc: 20, Food: map[string]int{"cam": 1}},
	{Thoc: 30, Items: map[string]int{"hat_cai": 2}},
	{Thoc: 40, Food: map[string]int{"cam": 2}},
	{Thoc: 50, Items: map[string]int{"hat_ca_rot": 2}},
	{Thoc: 60, Items: map[string]int{"hat_bi_ngo": 1}},
	{Thoc: 80, Food: map[string]int{"cam": 3}},
	{Thoc: 0, Items: map[string]int{"trung_dom": 1}},
}

func (a *Account) nextStreak(now time.Time) int {
	// Lùi một ngày theo lịch Việt Nam, không phụ thuộc giờ mùa hè (DST) của máy chủ.
	if a.CheckIn.Day == DayKey(now.In(vnZone).AddDate(0, 0, -1)) && a.CheckIn.Streak < len(CheckInRewards) {
		return a.CheckIn.Streak + 1
	}
	return 1
}

// DoCheckIn điểm danh hôm nay (mỗi ngày một lần), trả về ngày thứ mấy và phần quà.
func (a *Account) DoCheckIn(now time.Time) (int, Reward, error) {
	today := DayKey(now)
	if a.CheckIn.Day == today {
		return 0, Reward{}, errors.New("Hôm nay đã điểm danh rồi.")
	}
	day := a.nextStreak(now)
	rw := CheckInRewards[day-1]
	if a.addItems(rw.Items) != nil {
		// Kho đầy: vẫn điểm danh (không mất chuỗi ngày), vật phẩm vào hàng chờ như quà cấp (M12, PLAN.md 7.1).
		for id, n := range rw.Items {
			rw.Dropped += a.queuePending(id, n)
		}
		rw.Queued = true
	}
	for id, n := range rw.Food {
		a.Food[id] += n
	}
	if rw.Thoc > 0 {
		_ = a.addThoc(rw.Thoc, "check_in", today, now)
	}
	a.CheckIn = CheckIn{Day: today, Streak: day}
	return day, rw, nil
}

func (a *Account) CheckInView(now time.Time) CheckInView {
	if a.CheckIn.Day == DayKey(now) {
		next := a.CheckIn.Streak + 1
		if next > len(CheckInRewards) {
			next = 1
		}
		return CheckInView{Today: true, Streak: a.CheckIn.Streak, Next: next}
	}
	return CheckInView{Streak: a.CheckIn.Streak, Next: a.nextStreak(now)}
}
