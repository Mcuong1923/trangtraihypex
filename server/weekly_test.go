package main

import (
	"fmt"
	"testing"
	"time"
)

// Bảng xếp hạng tuần (PLAN_LOGIC_GAME 16): giữ kỷ lục cao nhất cho giun dài, cộng dồn giun hiếm và trận PK
// trong tuần, tối đa 10 người mỗi bảng; sang tuần mới thì chốt người đứng đầu và trao danh hiệu.

func TestWeeklyBoardsAndTitles(t *testing.T) {
	dir := t.TempDir()
	w := OpenWeekly(dir)
	mon := time.Date(2026, 11, 2, 9, 0, 0, 0, vnZone) // thứ Hai
	w.Submit("longest", "a", "Gà A", 12.5, "Giun đất", mon)
	w.Submit("longest", "b", "Gà B", 30, "Giun vàng", mon)
	w.Submit("longest", "a", "Gà A", 8, "Giun đất", mon) // kém kỷ lục cũ: giữ 12,5
	for i := 0; i < 15; i++ {
		w.Submit("rare", fmt.Sprintf("p%d", i), "Gà", float64(i), "", mon)
	}
	v := w.View(mon)
	boards := v["boards"].([]map[string]any)
	long := boards[0]["entries"].([]WeekEntry)
	if len(long) != 2 || long[0].ID != "b" || long[1].Value != 12.5 {
		t.Fatalf("bảng giun dài: %+v", long)
	}
	if rare := boards[1]["entries"].([]WeekEntry); len(rare) != WeeklyTop || rare[0].Value != 14 {
		t.Fatalf("bảng giun hiếm giữ 10 người cao nhất: %+v", rare)
	}
	if len(w.TitlesFor("b", mon)) != 0 {
		t.Fatal("chưa hết tuần thì chưa trao danh hiệu")
	}
	// Sang tuần sau: chốt người đứng đầu, bảng mới trống; dữ liệu còn sau khi mở lại.
	next := mon.AddDate(0, 0, 7)
	w2 := OpenWeekly(dir)
	got := w2.TitlesFor("b", next)
	if len(got) != 1 || got[0].Title != "vua_giun" {
		t.Fatalf("Gà B đứng đầu giun dài: %+v", got)
	}
	if v := w2.View(next); len(v["boards"].([]map[string]any)[0]["entries"].([]WeekEntry)) != 0 || v["last"] != weekKey(mon) {
		t.Fatal("tuần mới bảng trống, nhớ tuần vừa chốt")
	}
	// Bộ đếm tuần của tài khoản tự về 0 khi sang tuần.
	a := newAcc()
	a.week(mon).Rare = 3
	if a.week(next).Rare != 0 {
		t.Fatal("bộ đếm tuần phải về 0")
	}
}
