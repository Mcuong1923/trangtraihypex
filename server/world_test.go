package main

import "testing"

// Bản đồ liền (PLAN_THE_GIOI_3D mục 6, A04/A06): không còn cổng dịch chuyển; các khu không chồng nhau,
// đầu đường của mỗi khu đứng được và nằm ngoài khu; sảnh chỉ mở ra ngoài qua cửa chính.
func TestWorldTopology(t *testing.T) {
	for _, reg := range Regions {
		w := reg.World
		for id, z := range reg.Zones {
			for _, it := range z.Items {
				if it.Kind == "gate" {
					t.Fatalf("%s: không được còn cổng dịch chuyển %s", id, it.ID)
				}
			}
			for _, sp := range z.Spawns {
				if !w.Walkable(sp) || reg.AreaAt(sp) != id {
					t.Fatalf("%s: điểm xuất hiện %+v phải đứng được và thuộc khu", id, sp)
				}
			}
			if z.Outdoor {
				continue
			}
			if z.Entry == nil || !w.Walkable(*z.Entry) || reg.AreaAt(*z.Entry) != reg.Outdoor {
				t.Fatalf("%s: đầu đường %+v phải ở ngoài trời và đứng được", id, z.Entry)
			}
			for id2, z2 := range reg.Zones {
				if id2 != id && !z2.Outdoor && (z.HalfX+z2.HalfX > abs(z.CX-z2.CX)) && (z.HalfZ+z2.HalfZ > abs(z.CZ-z2.CZ)) {
					t.Fatalf("%s chồng lên %s", id, id2)
				}
			}
		}
	}
	lobby, w := Zones[ZoneLobby], Regions[RegionHypex].World
	n := 0
	for _, it := range lobby.Items {
		if it.Kind == "nest" {
			n++
			if !w.Walkable(it.Pos()) {
				t.Fatalf("ổ %s bị vật cản chặn", it.ID)
			}
		}
	}
	if n != 40 {
		t.Fatalf("sảnh cần 40 ổ, có %d", n)
	}
	at := func(x, z float64) Vec { return Vec{lobby.CX + x, lobby.CZ + z} }
	if !w.Walkable(at(0, -14.4)) || w.Walkable(at(0, -17)) {
		t.Fatal("bục chủ chuồng phải chặn lối, mép trước bục phải đứng được")
	}
	if !w.Walkable(at(0, 21)) || !w.Walkable(at(3, 21)) || w.Walkable(at(-8, 21)) || w.Walkable(at(-16, 0)) {
		t.Fatal("tường sảnh phải chặn, chỉ cửa chính đi qua được")
	}
	if a := Regions[RegionHypex].AreaAt(at(0, 25)); a != "village" {
		t.Fatalf("ra khỏi cửa sảnh phải là sân làng, đang là %s", a)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
