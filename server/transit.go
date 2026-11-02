package main

import "time"

// Lượt xe miễn phí (PLAN_LOGIC_GAME 7.2): tối đa 2 lượt, hồi 1 lượt mỗi TokenEvery theo giờ thực (cả khi offline).
// Dùng chung cho xe buýt vòng làng và xe liên trại.

const (
	MaxBusTokens = 2
	VillageFare  = 3 // vé xe buýt vòng làng khi hết lượt miễn phí
)

// refillTokens cộng các lượt đã hồi kể từ mốc BusTokenAt; đầy thì dừng đồng hồ hồi.
func (a *Account) refillTokens(now time.Time, b Balance) {
	if a.BusTokens >= MaxBusTokens {
		a.BusTokens, a.BusTokenAt = MaxBusTokens, time.Time{}
		return
	}
	if a.BusTokenAt.IsZero() {
		a.BusTokenAt = now
		return
	}
	if n := int(now.Sub(a.BusTokenAt) / b.TokenEvery); n > 0 {
		a.BusTokens = min(MaxBusTokens, a.BusTokens+n)
		a.BusTokenAt = a.BusTokenAt.Add(time.Duration(n) * b.TokenEvery)
		if a.BusTokens >= MaxBusTokens {
			a.BusTokenAt = time.Time{}
		}
	}
}

// Tokens là số lượt miễn phí đang có.
func (a *Account) Tokens(now time.Time, b Balance) int {
	a.refillTokens(now, b)
	return a.BusTokens
}

// NextTokenIn: số giây tới khi hồi lượt kế tiếp (0 khi đã đầy).
func (a *Account) NextTokenIn(now time.Time, b Balance) float64 {
	a.refillTokens(now, b)
	if a.BusTokens >= MaxBusTokens || a.BusTokenAt.IsZero() {
		return 0
	}
	return max(0, a.BusTokenAt.Add(b.TokenEvery).Sub(now).Seconds())
}

// useToken tiêu một lượt; lần đầu tiêu từ trạng thái đầy thì bắt đầu đồng hồ hồi.
func (a *Account) useToken(now time.Time, b Balance) bool {
	a.refillTokens(now, b)
	if a.BusTokens <= 0 {
		return false
	}
	if a.BusTokens >= MaxBusTokens {
		a.BusTokenAt = now
	}
	a.BusTokens--
	return true
}

// refundToken trả lại một lượt (huỷ vé trước giờ chạy).
func (a *Account) refundToken(now time.Time, b Balance) {
	a.refillTokens(now, b)
	a.BusTokens = min(MaxBusTokens, a.BusTokens+1)
	if a.BusTokens >= MaxBusTokens {
		a.BusTokenAt = time.Time{}
	}
}

// payRide trả một chuyến: ưu tiên lượt miễn phí, hết lượt thì trả thóc (đã áp "Mặc Cả"). Trả về (dùng lượt, thóc đã trả).
func (a *Account) payRide(fare int, reason, ref string, now time.Time, b Balance) (bool, int, error) {
	if a.useToken(now, b) {
		return true, 0, nil
	}
	cost := fareAfter(a, fare)
	if err := a.addThoc(-cost, reason, ref, now); err != nil {
		return false, 0, err
	}
	return false, cost, nil
}
