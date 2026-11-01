package main

import (
	"errors"
	"fmt"
	"math"
	"time"
)

// Phần V1.1/V1.2 chạy trong goroutine phòng: trạm ấp trứng, bến xe buýt, xe riêng chở bạn.

type Bus struct {
	To       string
	Trip     string
	DepartAt time.Time
}

// Ride là xe đang ra khỏi điểm đỗ; Seats[0] là người lái (chỉ chủ xe lái, mục 8.4).
type Ride struct {
	Kind  string
	Seats []*Player
}

type rideSpec struct {
	Kind   string
	Driver string // account ID người lái
	Seat   int
}

// Vị trí ghế so với người lái, theo hệ toạ độ của xe (trục z hướng về phía trước).
var seatOffsets = map[string][]Vec{
	"xe_dap": {{0, 0}, {0, -0.75}},
	"xe_may": {{0, 0}, {0, -0.75}},
	"o_to":   {{-0.45, 0.3}, {0.45, 0.3}, {-0.45, -0.7}, {0.45, -0.7}},
}

func (r *Room) stationZone() string { return Regions[r.Region].Station }

func (r *Room) otherRegion() string {
	if r.Region == RegionHypex {
		return RegionVenSuoi
	}
	return RegionHypex
}

func (r *Room) seatsBooked(to string) int {
	n := 0
	for _, p := range r.players {
		if bk := p.acc.Booking; bk != nil && bk.To == to && bk.State == "held" {
			n++
		}
	}
	return n
}

func (r *Room) busInfo() map[string]any {
	out := map[string]any{}
	for to, b := range r.buses {
		out[to] = map[string]any{"departIn": math.Max(0, time.Until(b.DepartAt).Seconds()), "seats": r.seatsBooked(to), "trip": b.Trip}
	}
	return out
}

func (r *Room) broadcastBus() {
	r.broadcastZone(r.stationZone(), nil, map[string]any{"t": "bus", "bus": r.busInfo()})
}

// cancelTrip huỷ vé chưa chạy và hoàn đúng một lần (rời bến, mất kết nối quá lâu, chuyến bị huỷ).
func (r *Room) cancelTrip(p *Player, why string) {
	if bk := p.acc.Booking; bk == nil || bk.State != "held" {
		return
	}
	msg, err := p.acc.CancelBooking(time.Now())
	if err != nil {
		return
	}
	r.hub.store.Save(p.acc)
	r.toast(p, "warn", why+" "+msg)
	p.sess.Send(map[string]any{"t": "me", "me": p.acc.View(time.Now(), r.hub.bal)})
}

// stepV11 chạy mỗi tick: xe buýt khởi hành, báo trứng nở, đặt khách đúng ghế.
func (r *Room) stepV11(now time.Time) {
	for to, bus := range r.buses {
		if now.Before(bus.DepartAt) {
			continue
		}
		delete(r.buses, to)
		var group []*Session
		var arrs []*arrival
		for _, p := range r.players {
			bk := p.acc.Booking
			if bk == nil || bk.To != to || bk.State != "held" {
				continue
			}
			if !p.goneAt.IsZero() || p.zone != r.stationZone() {
				r.cancelTrip(p, "Bạn không có mặt ở bến lúc xe chạy.")
				continue
			}
			// Chốt "đã chạy" xuống đĩa trước khi bàn giao: lỗi giữa chừng sẽ được đối soát khi vào lại.
			bk.State, bk.Trip = "departed", bus.Trip
			if err := r.hub.store.Save(p.acc); err != nil {
				bk.State, bk.Trip = "held", ""
				r.cancelTrip(p, "Máy chủ lỗi khi xuất bến.")
				continue
			}
			p.sess.Send(map[string]any{"t": "trip", "to": to, "title": Regions[to].Title})
			group = append(group, p.sess)
			arrs = append(arrs, &arrival{Zone: Regions[to].Station, Trip: bus.Trip})
		}
		if len(group) > 0 {
			r.system(fmt.Sprintf("🚌 Xe buýt chở %d khách rời bến đi %s.", len(group), Regions[to].Title))
			go r.hub.TransferRegion(group, to, arrs)
		}
		r.broadcastBus()
	}
	for _, p := range r.players {
		if inc := p.acc.Incubation; inc != nil && !inc.Notified && !now.Before(inc.DoneAt) {
			inc.Notified = true
			p.dirty = true
			r.toast(p, "good", "🐣 Trứng sắp nở! Gặp Vịt Bông ở Trạm ấp trứng.")
			id := p.acc.ID
			go func() {
				if r.hub.store.Social.AddMail(id, "hatch", "🐣 Trứng trong ổ ấp đã sẵn sàng nở. Gặp Vịt Bông để đón gà con!") {
					r.hub.mailPing(id)
				}
			}()
		}
	}
	r.syncRides()
	for _, p := range r.players {
		if t := p.trade; t != nil && !t.Open && now.After(t.Exp) {
			r.closeTrade(t, "Lời mời giao dịch hết hạn.")
		}
	}
	// Cộng tiền công/hoàn ký quỹ đang chờ, mỗi giây một lần.
	if r.tick%TickRate == 0 {
		for _, p := range r.players {
			if p.goneAt.IsZero() {
				r.applyCredits(p, now)
			}
		}
	}
}

// syncRides đặt khách theo ghế của xe: khách không gửi input đi bộ để dịch chuyển xe (mục 16.3).
func (r *Room) syncRides() {
	for _, p := range r.players {
		if p.ride == nil {
			continue
		}
		offs := seatOffsets[p.ride.Kind]
		s, c := math.Sin(p.ry), math.Cos(p.ry)
		for i, q := range p.ride.Seats {
			if i == 0 || q == nil {
				continue
			}
			o := Vec{offs[i].X - offs[0].X, offs[i].Z - offs[0].Z}
			q.pos = Vec{p.pos.X + o.X*c + o.Z*s, p.pos.Z - o.X*s + o.Z*c}
			q.ry = p.ry
			q.moving = p.moving
		}
	}
}

func (r *Room) rideOf(p *Player) *Player {
	if p.rideOf != nil {
		return p.rideOf
	}
	return nil
}

// dismount cho khách xuống ở vị trí an toàn cạnh xe, không mất đồ (mục 8.4).
func (r *Room) dismount(q *Player, why string) {
	d := q.rideOf
	if d == nil || d.ride == nil {
		q.rideOf = nil
		return
	}
	for i, x := range d.ride.Seats {
		if x == q {
			d.ride.Seats[i] = nil
		}
	}
	q.rideOf = nil
	q.pos = r.zones[q.zone].Clamp(Vec{q.pos.X + 1.2, q.pos.Z})
	if why != "" {
		r.toast(q, "warn", why)
	}
	r.broadcastRide(d)
	r.broadcastZone(q.zone, nil, map[string]any{"t": "pride", "pid": q.PID, "rideOf": ""})
}

func (r *Room) parkVehicle(p *Player, why string) {
	if p.ride == nil {
		return
	}
	for _, q := range p.ride.Seats[1:] {
		if q != nil {
			r.dismount(q, why)
		}
	}
	p.ride = nil
	r.broadcastZone(p.zone, nil, map[string]any{"t": "pveh", "pid": p.PID, "vehicle": "", "seats": []string{}})
}

func (r *Room) broadcastRide(d *Player) {
	seats := []string{}
	for _, q := range d.ride.Seats {
		if q == nil {
			seats = append(seats, "")
		} else {
			seats = append(seats, q.PID)
		}
	}
	r.broadcastZone(d.zone, nil, map[string]any{"t": "pveh", "pid": d.PID, "vehicle": d.ride.Kind, "seats": seats})
}

func freeSeat(rd *Ride) int {
	for i := 1; i < len(rd.Seats); i++ {
		if rd.Seats[i] == nil {
			return i
		}
	}
	return -1
}

// arriveV11 xử lý lúc vào phòng: điểm xuất hiện khi tới bằng xe, quyết toán vé, dựng lại xe đi liên vùng.
func (r *Room) arriveV11(p *Player, arr *arrival, now time.Time) {
	if arr != nil && arr.Zone != "" && r.zones[arr.Zone] != nil {
		p.zone = arr.Zone
	}
	if bk := p.acc.Booking; bk != nil {
		if bk.State == "departed" && arr != nil && arr.Trip == bk.Trip && bk.To == r.Region {
			p.acc.Booking = nil // tới nơi: hoàn tất chuyến
			p.dirty = true
		} else if bk.State == "held" {
			// Vé giữ chỗ không còn chuyến (máy chủ khởi động lại / phiên mới): hoàn đúng một lần.
			if msg, err := p.acc.CancelBooking(now); err == nil {
				r.hub.store.Save(p.acc)
				r.toast(p, "warn", "Chuyến xe trước chưa chạy. "+msg)
			}
		}
	}
	if arr != nil && arr.Ride != nil {
		rs := arr.Ride
		if rs.Seat == 0 {
			v := vehicleByID(rs.Kind)
			p.ride = &Ride{Kind: rs.Kind, Seats: make([]*Player, v.Seats)}
			p.ride.Seats[0] = p
		} else {
			for _, d := range r.players {
				if d.acc.ID == rs.Driver && d.ride != nil && rs.Seat < len(d.ride.Seats) && d.ride.Seats[rs.Seat] == nil {
					d.ride.Seats[rs.Seat] = p
					p.rideOf = d
				}
			}
		}
	}
}

func (r *Room) v11Act(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn func(func() (ActResult, error))) {
	now := time.Now()
	b := r.hub.bal
	zone := r.zones[p.zone]
	near := func(kind, msg string) bool {
		if nearKind(zone, p.pos, kind) == nil {
			fail(errors.New(msg))
			return false
		}
		return true
	}
	switch m.A {
	// ----- Trạm ấp trứng -----
	case "eggs_claim":
		if !near("npc_duck", "Gặp Vịt Bông ở Trạm ấp trứng.") {
			return
		}
		txn(func() (ActResult, error) {
			if n := p.acc.ClaimEggs(); n > 0 {
				return ActResult{Msg: fmt.Sprintf("Vịt Bông tặng bạn %d quả trứng 🥚", n)}, nil
			}
			return ActResult{}, fmt.Errorf("Chưa tới mốc nhận trứng (cấp %v).", EggMilestones)
		})
	case "incubate":
		if !near("npc_duck", "Gặp Vịt Bông để đặt trứng vào ổ ấp.") {
			return
		}
		txn(func() (ActResult, error) {
			inc, err := p.acc.StartIncubation(now, b)
			if err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: fmt.Sprintf("Đã đặt trứng vào ổ ấp — nở sau %s, kể cả khi bạn offline.", humanDur(inc.DoneAt.Sub(now)))}, nil
		})
	case "hatch":
		if !near("npc_duck", "Gặp Vịt Bông để đón gà con.") {
			return
		}
		txn(func() (ActResult, error) {
			c, err := p.acc.Hatch(m.ID, m.Text, r.rng.Float64(), now)
			if err != nil {
				return ActResult{}, err
			}
			r.broadcastLook(p)
			return ActResult{Msg: "🐣 Chào mừng " + c.Name + " ra đời!", Data: map[string]any{"pet": c}}, nil
		})
	case "follow", "rename_pet", "care":
		txn(func() (ActResult, error) {
			switch m.A {
			case "follow":
				if err := p.acc.Follow(m.ID); err != nil {
					return ActResult{}, err
				}
				r.broadcastLook(p)
				return ActResult{}, nil
			case "rename_pet":
				if err := p.acc.RenameCompanion(m.ID, m.Text); err != nil {
					return ActResult{}, err
				}
				r.broadcastLook(p)
				return ActResult{Msg: "Đã đổi tên."}, nil
			}
			gain, err := p.acc.Care(m.ID, m.Mode, now)
			if err != nil {
				return ActResult{}, err
			}
			r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "pet"})
			return ActResult{Msg: fmt.Sprintf("Độ thân thiết +%d", gain)}, nil
		})

	// ----- Bến xe -----
	case "book":
		if p.zone != r.stationZone() || !near("bus_stop", "Đặt vé ở Trạm xe buýt.") {
			return
		}
		if p.ride != nil || p.rideOf != nil {
			fail(errors.New("Cất xe trước khi đi xe buýt."))
			return
		}
		to := r.otherRegion()
		if r.seatsBooked(to) >= BusSeats {
			fail(fmt.Errorf("Chuyến này đã đủ %d ghế, chờ chuyến sau nhé.", BusSeats))
			return
		}
		txn(func() (ActResult, error) {
			bk, err := p.acc.Book(r.Region, to, now)
			if err != nil {
				return ActResult{}, err
			}
			if r.buses[to] == nil {
				r.buses[to] = &Bus{To: to, Trip: fmt.Sprintf("trip%d", now.UnixNano()), DepartAt: now.Add(b.BusWait)}
			}
			r.broadcastBus()
			how := "dùng 1 lượt miễn phí"
			if !bk.Free {
				how = fmt.Sprintf("trả %d thóc", bk.Paid)
			}
			return ActResult{Msg: fmt.Sprintf("Đã giữ ghế đi %s (%s). Xe chạy sau %.0f giây.", Regions[to].Title, how, time.Until(r.buses[to].DepartAt).Seconds())}, nil
		})
	case "cancel_booking":
		txn(func() (ActResult, error) {
			msg, err := p.acc.CancelBooking(now)
			r.broadcastBus()
			return ActResult{Msg: msg}, err
		})

	// ----- Xe riêng -----
	case "vehicle_buy":
		if !near("npc_goat", "Mua xe ở Cửa hàng xe.") {
			return
		}
		txn(func() (ActResult, error) {
			if err := p.acc.BuyVehicle(m.ID, now); err != nil {
				return ActResult{}, err
			}
			return ActResult{Msg: "Đã mua " + vehicleByID(m.ID).Name + "! Lấy xe ở điểm đỗ."}, nil
		})
	case "vehicle_out":
		v := vehicleByID(m.ID)
		switch {
		case !near("parking", "Lấy xe ở điểm đỗ."):
			return
		case v == nil || !p.acc.OwnedVehicles[m.ID]:
			fail(errors.New("Bạn chưa có xe này."))
		case p.ride != nil || p.rideOf != nil:
			fail(errors.New("Mỗi người tối đa một xe đang ra."))
		case p.match != nil || p.acc.HungerStage(b) >= 4:
			fail(errors.New("Đang kiệt sức hoặc trong trận, chưa lái được."))
		default:
			p.ride = &Ride{Kind: v.ID, Seats: make([]*Player, v.Seats)}
			p.ride.Seats[0] = p
			r.broadcastRide(p)
			reply(ActResult{OK: true, Msg: "Đã lấy " + v.Name + ". Mời bạn lên xe bằng nút Mời lên xe."})
		}
	case "vehicle_in":
		if p.ride == nil {
			fail(errors.New("Bạn không có xe đang ra."))
			return
		}
		if !near("parking", "Cất xe ở điểm đỗ.") {
			return
		}
		r.parkVehicle(p, "Chủ xe đã cất xe, bạn xuống ở điểm đỗ.")
		reply(ActResult{OK: true, Msg: "Đã cất xe."})
	case "seat_invite":
		t := r.byPID(m.Target)
		switch {
		case p.ride == nil:
			fail(errors.New("Lấy xe ra trước đã."))
		case t == nil || t == p || t.zone != p.zone || t.pos.Dist(p.pos) > 4:
			fail(errors.New("Người được mời phải đứng gần xe."))
		case t.ride != nil || t.rideOf != nil || t.match != nil:
			fail(errors.New(t.acc.Char.Name + " đang bận."))
		case freeSeat(p.ride) < 0:
			fail(errors.New("Xe đã hết ghế."))
		case r.hub.store.Social.Blocked(p.acc.ID, t.acc.ID):
			fail(ErrBlocked)
		default:
			t.seatInvites[p.PID] = now.Add(20 * time.Second)
			t.sess.Send(map[string]any{"t": "seat_invite", "from": p.acc.Char.Name, "driver": p.PID, "vehicle": vehicleByID(p.ride.Kind).Name})
			reply(ActResult{OK: true, Msg: "Đã mời " + t.acc.Char.Name + " lên xe."})
		}
	case "seat_accept":
		d := r.byPID(m.Target)
		exp, ok := p.seatInvites[m.Target]
		delete(p.seatInvites, m.Target)
		switch {
		case !ok || now.After(exp) || d == nil || d.ride == nil:
			fail(errors.New("Lời mời lên xe đã hết hạn."))
		case d.zone != p.zone || d.pos.Dist(p.pos) > 4:
			fail(errors.New("Hãy lại gần xe."))
		case p.ride != nil || p.rideOf != nil || p.match != nil:
			fail(errors.New("Bạn đang bận."))
		default:
			// Ghế là ràng buộc máy chủ: hai người tranh ghế cuối thì chỉ một người được nhận.
			i := freeSeat(d.ride)
			if i < 0 {
				fail(errors.New("Xe vừa hết ghế."))
				return
			}
			d.ride.Seats[i] = p
			p.rideOf = d
			p.dig = nil
			r.broadcastRide(d)
			r.broadcastZone(p.zone, nil, map[string]any{"t": "pride", "pid": p.PID, "rideOf": d.PID})
			reply(ActResult{OK: true, Msg: "Đã lên xe của " + d.acc.Char.Name})
		}
	case "seat_leave":
		if p.rideOf == nil {
			fail(errors.New("Bạn không ngồi trên xe nào."))
			return
		}
		r.dismount(p, "")
		reply(ActResult{OK: true})
	case "seat_kick":
		t := r.byPID(m.Target)
		if p.ride == nil || t == nil || t.rideOf != p {
			fail(errors.New("Người này không ngồi trên xe của bạn."))
			return
		}
		r.dismount(t, "Chủ xe đã mời bạn xuống.")
		reply(ActResult{OK: true})
	case "drive_region":
		v := p.ride
		switch {
		case v == nil:
			fail(errors.New("Lấy xe ra trước đã."))
			return
		case !vehicleByID(v.Kind).Road:
			fail(errors.New("Xe đạp chỉ đi trong trại; đi liên trại cần xe máy hoặc ô tô."))
			return
		case !near("road", "Đi tới cổng Đường liên trại."):
			return
		}
		to := r.otherRegion()
		group := []*Session{p.sess}
		arrs := []*arrival{{Zone: Regions[to].Station, Ride: &rideSpec{Kind: v.Kind, Driver: p.acc.ID}}}
		for i, q := range v.Seats[1:] {
			if q == nil {
				continue
			}
			if q.acc.Level < Regions[to].Level {
				// Mỗi người phải đủ điều kiện vào vùng; không chở người chưa mở vùng vượt khoá cấp (mục 8.4).
				fail(fmt.Errorf("%s cần cấp %d để tới %s.", q.acc.Char.Name, Regions[to].Level, Regions[to].Title))
				return
			}
			group = append(group, q.sess)
			arrs = append(arrs, &arrival{Zone: Regions[to].Station, Ride: &rideSpec{Kind: v.Kind, Driver: p.acc.ID, Seat: i + 1}})
		}
		if p.acc.Level < Regions[to].Level {
			fail(fmt.Errorf("Cần cấp %d để tới %s.", Regions[to].Level, Regions[to].Title))
			return
		}
		reply(ActResult{OK: true, Msg: "Lên đường tới " + Regions[to].Title + "!"})
		for _, s := range group {
			s.Send(map[string]any{"t": "trip", "to": to, "title": Regions[to].Title, "vehicle": v.Kind})
		}
		go r.hub.TransferRegion(group, to, arrs)
	}
}
