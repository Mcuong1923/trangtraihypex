package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Kho đồ (PLAN_LOGIC_GAME mục 11–12): vật liệu xây nhà, hạt giống, nông sản, trứng. Thức ăn mua ở chợ
// vẫn nằm trong Account.Food như trước; giao diện gộp chung một tab "Kho".

type ItemDef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emo   string `json:"emo"`
	Group string `json:"group"` // vatlieu | hatgiong | nongsan | trung
	Buy   int    `json:"buy"`   // giá ở tiệm (0: không bán ở tiệm)
	Sell  int    `json:"sell"`  // giá thu lại ở quầy thức ăn (0: không thu)
	Shop  string `json:"shop"`  // vatlieu (Bò Mộc) | thucan (quầy thức ăn) | ""
	Mana  int    `json:"mana"`  // nông sản ăn được
	Trade bool   `json:"trade"` // được trao đổi giữa người chơi (từ cấp TradeLevel)
}

var Items = []ItemDef{
	{"rom", "Rơm", "🌾", "vatlieu", 3, 0, "vatlieu", 0, true},
	{"go", "Gỗ", "🪵", "vatlieu", 8, 0, "vatlieu", 0, true},
	{"dinh", "Đinh", "🔩", "vatlieu", 2, 0, "vatlieu", 0, true},
	{"da", "Đá", "🪨", "vatlieu", 6, 0, "vatlieu", 0, true},
	{"gach", "Gạch", "🧱", "vatlieu", 12, 0, "vatlieu", 0, true},
	{"ngoi", "Ngói", "🏠", "vatlieu", 10, 0, "vatlieu", 0, true},
	{"hat_cai", "Hạt cải", "🌱", "hatgiong", 3, 0, "thucan", 0, true},
	{"hat_ca_rot", "Hạt cà rốt", "🌱", "hatgiong", 5, 0, "thucan", 0, true},
	{"hat_bi_ngo", "Hạt bí ngô", "🌱", "hatgiong", 12, 0, "thucan", 0, true},
	{"hat_hoa", "Hạt hoa", "🌱", "hatgiong", 6, 0, "thucan", 0, true},
	{"cai", "Cải", "🥬", "nongsan", 0, 4, "", 15, true},
	{"ca_rot", "Cà rốt", "🥕", "nongsan", 0, 7, "", 25, true},
	{"bi_ngo", "Bí ngô", "🎃", "nongsan", 0, 25, "", 80, true},
	{"bo_hoa", "Bó hoa", "💐", "nongsan", 0, 8, "", 0, true},
	{"phan_u", "Phân ủ", "🟤", "nongsan", 0, 0, "", 0, false},
	// Cá câu ở ao (PLAN_LOGIC_GAME 16): bán ở Quầy thức ăn hoặc ăn hồi độ no.
	{"ca_ro", "Cá rô", "🐟", "ca", 0, 6, "", 18, true},
	{"ca_chep", "Cá chép", "🐠", "ca", 0, 14, "", 30, true},
	{"ca_tre", "Cá trê", "🐡", "ca", 0, 35, "", 50, true},
	{"ca_vang", "Cá vàng", "🌟", "ca", 0, 120, "", 90, true},
	{"trung_thuong", "Trứng thường", "🥚", "trung", 0, 0, "", 0, false},
	{"trung_dom", "Trứng đốm", "🥚", "trung", 0, 0, "", 0, false},
	{"trung_vang", "Trứng vàng", "🥚", "trung", 0, 0, "", 0, false},
}

// StorageCap: kho chứa tối đa 200 đơn vị vật liệu, hạt giống và nông sản (trứng không tính).
const StorageCap = 200

func itemByID(id string) *ItemDef {
	for i := range Items {
		if Items[i].ID == id {
			return &Items[i]
		}
	}
	return nil
}

func (a *Account) StorageCount() int {
	n := 0
	for id, c := range a.Items {
		if it := itemByID(id); it != nil && it.Group != "trung" {
			n += c
		}
	}
	return n
}

// addItems cộng vật phẩm; vượt sức chứa kho thì báo lỗi và không đổi gì.
func (a *Account) addItems(add map[string]int) error {
	extra := 0
	for id, n := range add {
		it := itemByID(id)
		if it == nil {
			return fmt.Errorf("vật phẩm lạ %q", id)
		}
		if it.Group != "trung" {
			extra += n
		}
	}
	if extra > 0 && a.StorageCount()+extra > StorageCap {
		return fmt.Errorf("Kho đầy (%d/%d). Dùng bớt vật liệu, hạt giống hoặc nông sản.", a.StorageCount(), StorageCap)
	}
	for id, n := range add {
		a.Items[id] += n
	}
	return nil
}

// takeItems trừ đúng số lượng nếu đủ cả; thiếu một món thì không trừ món nào.
func (a *Account) takeItems(need map[string]int) error {
	var miss []string
	for id, n := range need {
		if a.Items[id] < n {
			miss = append(miss, fmt.Sprintf("%d %s", n-a.Items[id], itemByID(id).Name))
		}
	}
	if len(miss) > 0 {
		sort.Strings(miss)
		return errors.New("Còn thiếu " + strings.Join(miss, ", ") + ".")
	}
	for id, n := range need {
		a.Items[id] -= n
	}
	return nil
}

func (a *Account) eggCount() int {
	return a.Items["trung_thuong"] + a.Items["trung_dom"] + a.Items["trung_vang"]
}

// ---------- Tiệm vật liệu (Bò Mộc) và quầy hạt giống / thu nông sản (chợ) ----------

func (a *Account) BuyItem(id string, n int, shop string, now time.Time) error {
	it := itemByID(id)
	if it == nil || it.Buy <= 0 || it.Shop != shop {
		return errors.New("Tiệm này không bán món đó.")
	}
	if n < 1 || n > 50 {
		return errors.New("Mỗi lần mua từ 1 tới 50.")
	}
	if a.Thoc < it.Buy*n {
		return ErrNotEnough
	}
	if err := a.addItems(map[string]int{id: n}); err != nil {
		return err
	}
	return a.addThoc(-it.Buy*n, "buy_item", id, now)
}

func (a *Account) SellItem(id string, n int, now time.Time) (int, error) {
	it := itemByID(id)
	if it == nil || it.Sell <= 0 {
		return 0, errors.New("Quầy không thu món này.")
	}
	if n < 1 || a.Items[id] < n {
		return 0, errors.New("Không đủ " + it.Name + " trong kho.")
	}
	a.Items[id] -= n
	got := it.Sell * n
	return got, a.addThoc(got, "sell_item", id, now)
}

// EatItem ăn nông sản tự trồng (cải, cà rốt, bí ngô).
func (a *Account) EatItem(id string) (int, error) {
	it := itemByID(id)
	if it == nil || it.Mana <= 0 || a.Items[id] <= 0 {
		return 0, errors.New("Không có món này để ăn.")
	}
	if a.Mana >= ManaMax {
		return 0, errors.New("Bụng đã no căng, chưa ăn được.")
	}
	a.Items[id]--
	a.restoreMana(float64(it.Mana))
	a.step("first_meal")
	return it.Mana, nil
}

// ---------- Phiếu giảm giá (Dê Thợ Máy tặng khách mới) ----------

const CouponFirstBike = "xe_dau_tien" // giảm 10% chiếc xe đầu tiên, hạn 24 giờ

func (a *Account) GiveCoupon(id string, ttl time.Duration, now time.Time) bool {
	if _, had := a.Coupons[id]; had {
		return false // mỗi tài khoản nhận một lần, kể cả khi đã hết hạn
	}
	a.Coupons[id] = now.Add(ttl)
	return true
}

func (a *Account) couponLive(id string, now time.Time) bool {
	exp, ok := a.Coupons[id]
	return ok && now.Before(exp)
}

// ---------- Quà lên cấp ----------

type Reward struct {
	Thoc   int            `json:"thoc"`
	Items  map[string]int `json:"items,omitempty"`
	Food   map[string]int `json:"food,omitempty"`
	Title  string         `json:"title,omitempty"`
	Unlock string         `json:"unlock,omitempty"`
}

// LevelRewards: bảng mục 12. Mốc trứng cũ (cấp 5/7/9) nằm trong bảng; ai đã nhận qua Vịt Bông thì bỏ phần trứng.
var LevelRewards = map[int]Reward{
	2:  {Thoc: 50, Food: map[string]int{"cam": 3}, Items: map[string]int{"hat_cai": 2}},
	3:  {Thoc: 80, Items: map[string]int{"rom": 10, "go": 5, "hat_ca_rot": 2}, Unlock: "Mua đất, giao dịch"},
	4:  {Thoc: 100, Food: map[string]int{"rau": 2}, Items: map[string]int{"trung_thuong": 1}},
	5:  {Thoc: 120, Items: map[string]int{"go": 10, "da": 5, "trung_thuong": 1}, Unlock: "Xe máy"},
	6:  {Thoc: 150, Items: map[string]int{"hat_bi_ngo": 2, "dinh": 10}, Unlock: "Lô đất rộng"},
	7:  {Thoc: 180, Items: map[string]int{"gach": 10, "trung_thuong": 1}, Unlock: "Ô tô, nghề thợ xây"},
	8:  {Thoc: 200, Food: map[string]int{"suat": 2}, Items: map[string]int{"trung_dom": 1}},
	9:  {Thoc: 250, Items: map[string]int{"ngoi": 10, "gach": 10, "trung_thuong": 1}},
	10: {Thoc: 300, Items: map[string]int{"trung_vang": 1}, Title: "lao_lang", Unlock: "Lập Hội Gà"},
	// Cấp 11–20 (PLAN_LOGIC_GAME 16): vật liệu cho nhà lớn, trứng hiếm, lô đồi và nhà gạch hai tầng.
	11: {Thoc: 320, Items: map[string]int{"go": 15, "dinh": 15}},
	12: {Thoc: 350, Items: map[string]int{"da": 10, "gach": 10}, Unlock: "Lô đồi"},
	13: {Thoc: 380, Items: map[string]int{"trung_dom": 1, "hat_bi_ngo": 3}},
	14: {Thoc: 400, Food: map[string]int{"suat": 3}, Items: map[string]int{"ngoi": 10}},
	15: {Thoc: 450, Items: map[string]int{"gach": 20, "ngoi": 10}, Unlock: "Nhà gạch hai tầng"},
	16: {Thoc: 480, Items: map[string]int{"trung_vang": 1}},
	17: {Thoc: 520, Items: map[string]int{"go": 20, "gach": 10}},
	18: {Thoc: 560, Items: map[string]int{"trung_dom": 1, "hat_hoa": 3}},
	19: {Thoc: 600, Items: map[string]int{"gach": 20, "ngoi": 15}},
	20: {Thoc: 700, Items: map[string]int{"trung_vang": 1}, Title: "truong_lang", Unlock: "Danh hiệu Trưởng Làng"},
}

// ClaimLevelRewards trao quà mọi cấp đã đạt mà chưa nhận (idempotent theo cấp). Kho đầy thì phần vật phẩm
// vượt sức chứa bỏ qua nhưng thóc, thức ăn và trứng vẫn nhận, cấp vẫn đánh dấu để không nhận lặp.
func (a *Account) ClaimLevelRewards(now time.Time) map[int]Reward {
	got := map[int]Reward{}
	for lv := 2; lv <= a.Level; lv++ {
		rw, ok := LevelRewards[lv]
		if !ok || a.LevelRewarded[lv] {
			continue
		}
		a.LevelRewarded[lv] = true
		give := Reward{Thoc: rw.Thoc, Food: rw.Food, Title: rw.Title, Unlock: rw.Unlock, Items: map[string]int{}}
		for id, n := range rw.Items {
			if strings.HasPrefix(id, "trung_") && a.EggsClaimed[lv] {
				continue // đã nhận trứng mốc này qua Vịt Bông trước bản cập nhật
			}
			give.Items[id] = n
		}
		if a.addItems(give.Items) != nil {
			eggs := map[string]int{}
			for id, n := range give.Items {
				if strings.HasPrefix(id, "trung_") {
					eggs[id] = n
				}
			}
			give.Items = eggs
			_ = a.addItems(eggs)
		}
		for id, n := range rw.Food {
			a.Food[id] += n
		}
		_ = a.addThoc(rw.Thoc, "level_reward", fmt.Sprintf("lv%d", lv), now)
		if rw.Title != "" && !contains(a.Titles, rw.Title) {
			a.Titles = append(a.Titles, rw.Title)
		}
		got[lv] = give
	}
	return got
}

// rewardText tóm tắt quà cho toast/ghi chú.
// itemsText: "4 Gỗ, 1 Trứng thường" theo thứ tự mã cho ổn định.
func itemsText(items map[string]int) string {
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d %s", items[id], itemByID(id).Name))
	}
	return strings.Join(parts, ", ")
}

func rewardText(rw Reward) string {
	parts := []string{fmt.Sprintf("+%d thóc", rw.Thoc)}
	for id, n := range rw.Food {
		if f := foodByID(id); f != nil {
			parts = append(parts, fmt.Sprintf("%d %s", n, f.Name))
		}
	}
	if t := itemsText(rw.Items); t != "" {
		parts = append(parts, t)
	}
	if rw.Title != "" {
		parts = append(parts, "danh hiệu "+titleName(rw.Title))
	}
	return strings.Join(parts, ", ")
}

// levelUp trao quà các cấp mới, báo cho người chơi và để Cụ Cục chúc mừng.
func (r *Room) levelUp(p *Player, gained int, now time.Time, notes *[]string) {
	if gained <= 0 {
		return
	}
	got := p.acc.ClaimLevelRewards(now)
	lvs := make([]int, 0, len(got))
	for lv := range got {
		lvs = append(lvs, lv)
	}
	sort.Ints(lvs)
	for _, lv := range lvs {
		*notes = append(*notes, fmt.Sprintf("🎁 Quà cấp %d: %s", lv, rewardText(got[lv])))
		p.sess.Send(map[string]any{"t": "reward", "level": lv, "reward": got[lv]})
	}
	r.npcEvent(p, "levelup", nil)
}

// ---------- Phần thưởng phụ khi đào ----------

// digBonus: ụ "Gốc cây" có 8% rơi thêm 1 Gỗ, "Đất ẩm"/"Bùn ven suối" 6% thêm 1 Đá (không đụng bảng giun);
// bạn đồng hành "May Mắn Kép" có cơ hội cho thêm 1 con cùng loài.
func (r *Room) digBonus(p *Player, d *DigSession, now time.Time, notes *[]string) map[string]int {
	out := map[string]int{}
	soil := ""
	if it := r.zones[p.zone].Item(d.Spot); it != nil {
		soil = it.Label
	}
	switch {
	case soil == "Gốc cây" && r.rng.Float64() < 0.08:
		out["go"] = 1
	case (soil == "Đất ẩm" || soil == "Bùn ven suối") && r.rng.Float64() < 0.06:
		out["da"] = 1
	}
	if len(out) > 0 {
		if p.acc.addItems(out) != nil {
			out = map[string]int{}
		} else {
			for id := range out {
				*notes = append(*notes, "Đào trúng thêm 1 "+itemByID(id).Name+"!")
			}
		}
	}
	if ch := petBonus(p.acc, AbilityDouble); ch > 0 && r.rng.Float64() < ch && p.acc.BagCount() < BagCapacity {
		p.acc.Worms[d.Worm.ID]++
		out["worm:"+d.Worm.ID] = 1
		*notes = append(*notes, "🦚 May Mắn Kép: thêm 1 "+d.Worm.Name+"!")
	}
	return out
}
