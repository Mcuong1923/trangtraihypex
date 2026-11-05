package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Hạt giống và trồng cây (PLAN_LOGIC_GAME mục 14). Luống ở Ao và vườn (12 luống, mỗi người giữ tối đa 1)
// và luống rau trong nhà riêng. Cây là tài sản riêng của từng người (lưu trong tài khoản), lớn theo giờ thực
// cả khi offline, nên không ai hái được cây của người khác.

type Plant struct {
	Seed    string    `json:"seed"`
	At      time.Time `json:"at"`
	Ready   time.Time `json:"ready"`
	Wilt    time.Time `json:"wilt"`
	Watered bool      `json:"watered"`
	Fertile bool      `json:"fertile"` // đã bón phân ủ: thêm 1 nông sản
}

type SeedDef struct {
	Seed  string        `json:"seed"`
	Crop  string        `json:"crop"`
	Grow  time.Duration `json:"-"`
	Sec   float64       `json:"sec"`
	Yield int           `json:"yield"`
}

var Seeds = []SeedDef{
	{"hat_cai", "cai", 10 * time.Minute, 600, 3},
	{"hat_ca_rot", "ca_rot", 20 * time.Minute, 1200, 3},
	{"hat_bi_ngo", "bi_ngo", 45 * time.Minute, 2700, 1},
	{"hat_hoa", "bo_hoa", 30 * time.Minute, 1800, 1},
}

const (
	ParkBedHold = 2 * time.Hour // luống cộng đồng giữ tối đa 2 giờ
	WaterCut    = 0.25          // tưới: rút 25% thời gian còn lại, mỗi lần trồng tưới một lần
)

func seedByID(id string) *SeedDef {
	for i := range Seeds {
		if Seeds[i].Seed == id {
			return &Seeds[i]
		}
	}
	return nil
}

func isParkBed(key string) bool { return strings.HasPrefix(key, "park:") }

// PlantSeed gieo hạt vào luống. growScale < 1 chỉ dùng cho bộ cân bằng thử nhanh.
func (a *Account) PlantSeed(bed, seed string, fertilize bool, now time.Time, growScale float64) (*Plant, error) {
	sd := seedByID(seed)
	switch {
	case sd == nil:
		return nil, errors.New("Hạt giống không hợp lệ.")
	case a.Items[seed] <= 0:
		return nil, errors.New("Bạn không còn " + itemByID(seed).Name + ". Mua ở Quầy thức ăn.")
	case a.Beds[bed] != nil:
		return nil, errors.New("Luống này đang có cây.")
	case fertilize && a.Items["phan_u"] <= 0:
		return nil, errors.New("Bạn chưa có phân ủ.")
	}
	if isParkBed(bed) {
		for k := range a.Beds {
			if isParkBed(k) {
				return nil, errors.New("Mỗi người chỉ giữ một luống cộng đồng. Thu hoạch luống cũ trước.")
			}
		}
	}
	grow := time.Duration(float64(sd.Grow) * growScale)
	wilt := 3 * grow
	if isParkBed(bed) && wilt > ParkBedHold {
		wilt = ParkBedHold
	}
	a.Items[seed]--
	if fertilize {
		a.Items["phan_u"]--
	}
	pl := &Plant{Seed: seed, At: now, Ready: now.Add(grow), Wilt: now.Add(max(wilt, grow+time.Minute)), Fertile: fertilize}
	a.Beds[bed] = pl
	return pl, nil
}

func (a *Account) WaterBed(bed string, now time.Time) error {
	pl := a.Beds[bed]
	switch {
	case pl == nil:
		return errors.New("Luống này chưa có cây của bạn.")
	case pl.Watered:
		return errors.New("Đã tưới rồi, để cây nghỉ.")
	case !now.Before(pl.Ready):
		return errors.New("Cây chín rồi, thu hoạch thôi!")
	}
	pl.Watered = true
	pl.Ready = pl.Ready.Add(-time.Duration(float64(pl.Ready.Sub(now)) * WaterCut))
	return nil
}

// Harvest thu cây chín; để quá hạn thì cây héo, chỉ còn 1 phân ủ.
func (a *Account) Harvest(bed string, now time.Time) (string, int, error) {
	pl := a.Beds[bed]
	if pl == nil {
		return "", 0, errors.New("Luống này chưa có cây của bạn.")
	}
	if now.Before(pl.Ready) {
		return "", 0, fmt.Errorf("Cây chưa chín, còn %s.", humanDur(pl.Ready.Sub(now)))
	}
	sd := seedByID(pl.Seed)
	crop, n := sd.Crop, sd.Yield
	if pl.Fertile {
		n++
	}
	if now.After(pl.Wilt) {
		crop, n = "phan_u", 1
	}
	if err := a.addItems(map[string]int{crop: n}); err != nil {
		return "", 0, err
	}
	delete(a.Beds, bed)
	return crop, n, nil
}

// ---------- Tưới giúp bạn bè (PLAN_LOGIC_GAME 14) ----------
// Cây là tài sản trong tài khoản người trồng, nên chỉ tưới giúp được khi người bạn ấy đang ở cùng phòng
// (cùng kênh ở Ao và vườn, hoặc đang ở nhà khi mình sang thăm): phòng giữ tài khoản của cả hai người.

// friendBeds: cây của bạn bè cùng phòng mà người xem nhìn thấy được (luống cộng đồng; luống nhà khi đang ở
// trong nhà của chính bạn ấy).
func (r *Room) friendBeds(v *Player) []map[string]any {
	out := []map[string]any{}
	for _, q := range r.players {
		if q == v || q.acc.Char == nil || !q.goneAt.IsZero() || !r.hub.store.Social.AreFriends(v.acc.ID, q.acc.ID) {
			continue
		}
		for key, pl := range q.acc.Beds {
			home := strings.HasPrefix(key, "home:")
			if home && (r.home == nil || r.Owner != q.acc.ID) || !home && r.zones["park"] == nil {
				continue
			}
			out = append(out, map[string]any{"pid": q.PID, "name": q.acc.Char.Name, "key": key, "seed": pl.Seed,
				"at": pl.At, "ready": pl.Ready, "wilt": pl.Wilt, "watered": pl.Watered})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i]["pid"].(string)+out[i]["key"].(string) < out[j]["pid"].(string)+out[j]["key"].(string)
	})
	return out
}

// syncFriendBeds gửi lại danh sách cây của bạn bè cho ai có thay đổi (gọi mỗi giây khi phòng đánh dấu bẩn).
func (r *Room) syncFriendBeds() {
	if !r.fbedsDirty {
		return
	}
	r.fbedsDirty = false
	for _, v := range r.players {
		list := r.friendBeds(v)
		b, _ := json.Marshal(list)
		if string(b) == v.fbedsSent {
			continue
		}
		v.fbedsSent = string(b)
		v.sess.Send(map[string]any{"t": "fbeds", "beds": list})
	}
}

// friendBedKey kiểm tra người tưới đứng cạnh luống có cây của bạn q.
func (r *Room) friendBedKey(p, q *Player, id string) (string, error) {
	if strings.HasPrefix(id, "home:") {
		if r.home == nil || r.Owner != q.acc.ID {
			return "", errors.New("Chỉ tưới giúp luống trong nhà khi chủ nhà đang ở nhà.")
		}
		oid := strings.TrimPrefix(id, "home:")
		for _, o := range r.home.Objects {
			if o.ID == oid && (o.Kind == "luong_rau" || o.Kind == "luong_ca_rot") {
				if p.pos.Dist(Vec{float64(o.X), float64(o.Z)}) > 2.4 {
					return "", errors.New("Lại gần luống rau hơn.")
				}
				return id, nil
			}
		}
		return "", errors.New("Không tìm thấy luống rau này.")
	}
	zone := r.zones[p.zone]
	it := zone.Item(id)
	if it == nil || it.Kind != "bed" || p.pos.Dist(it.Pos()) > it.R+InteractSlack {
		return "", errors.New("Lại gần luống của bạn ấy ở Ao và vườn.")
	}
	return "park:" + id, nil
}
