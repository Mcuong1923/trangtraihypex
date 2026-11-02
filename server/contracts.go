package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Hợp đồng thuê người chơi xây chuồng theo mẫu (mục 9.2, V1.1). Thóc của chủ được giữ trong ký quỹ,
// máy chủ tự nghiệm thu khi đủ tiến độ, quá hạn thì hoàn chủ đúng một lần.
// Tiền trả/hoàn đi qua hàng chờ "credit" bền vững: phòng đang giữ tài khoản người nhận sẽ cộng vào ví.

type Contract struct {
	ID         string         `json:"id"`
	Owner      string         `json:"owner"`
	OwnerName  string         `json:"ownerName"`
	Worker     string         `json:"worker,omitempty"`
	WorkerName string         `json:"workerName,omitempty"`
	Roof       string         `json:"roof"`
	Material   string         `json:"material,omitempty"` // vật liệu chủ đã gom (PLAN_LOGIC_GAME 13.3)
	Items      map[string]int `json:"items,omitempty"`    // vật liệu đang giữ cùng ký quỹ, hoàn lại nếu huỷ/quá hạn
	Price      int            `json:"price"`
	State      string         `json:"state"` // open | taken | done | expired | cancelled
	Progress   int            `json:"progress"`
	Need       int            `json:"need"`
	CreatedAt  time.Time      `json:"createdAt"`
	Deadline   time.Time      `json:"deadline"`
	LastWork   time.Time      `json:"lastWork"`
}

type Credit struct {
	Items  map[string]int `json:"items,omitempty"` // vật liệu hoàn lại
	Amount int            `json:"amount"`
	Reason string         `json:"reason"`
	Ref    string         `json:"ref"`
	Note   string         `json:"note"`
}

type ContractStore struct {
	mu        sync.Mutex
	path      string
	Contracts map[string]*Contract `json:"contracts"`
	Credits   map[string][]Credit  `json:"credits"`
}

const (
	ContractMinPrice = 200
	ContractMaxPrice = 5000
	WorkGap          = 1500 * time.Millisecond // một lần thi công mất ~1,5 giây thao tác
)

func OpenContracts(dir string) (*ContractStore, error) {
	c := &ContractStore{path: filepath.Join(dir, "contracts.json"), Contracts: map[string]*Contract{}, Credits: map[string][]Credit{}}
	if b, err := os.ReadFile(c.path); err == nil {
		if err := json.Unmarshal(b, c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *ContractStore) saveLocked() error { return writeAtomic(c.path, c) }

// SetMaterials ghi vật liệu chủ đã giữ cho hợp đồng (hoàn lại khi huỷ/quá hạn).
func (c *ContractStore) SetMaterials(id, material string, items map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k := c.Contracts[id]; k != nil {
		k.Material, k.Items = material, items
		c.saveLocked()
	}
}

func (c *ContractStore) creditLocked(id string, cr Credit) {
	c.Credits[id] = append(c.Credits[id], cr)
}

// TakeCredits lấy (và xoá) các khoản chờ cộng cho tài khoản; gọi trong goroutine phòng đang giữ tài khoản.
func (c *ContractStore) TakeCredits(id string) []Credit {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := c.Credits[id]
	if len(l) == 0 {
		return nil
	}
	delete(c.Credits, id)
	if err := c.saveLocked(); err != nil {
		c.Credits[id] = l
		return nil
	}
	return l
}

// Restore trả lại khoản chưa cộng được (lưu tài khoản lỗi) để lần sau thử lại, không mất tiền.
func (c *ContractStore) Restore(id string, l []Credit) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Credits[id] = append(l, c.Credits[id]...)
	_ = c.saveLocked()
}

func (c *ContractStore) active(owner string) *Contract {
	for _, k := range c.Contracts {
		if k.Owner == owner && (k.State == "open" || k.State == "taken") {
			return k
		}
	}
	return nil
}

func (c *ContractStore) Active(owner string) *Contract {
	c.mu.Lock()
	defer c.mu.Unlock()
	if k := c.active(owner); k != nil {
		cp := *k
		return &cp
	}
	return nil
}

// Create ghi hợp đồng sau khi phòng đã trừ ký quỹ khỏi ví chủ.
func (c *ContractStore) Create(owner, ownerName, roof string, price int, need int, ttl time.Duration) (*Contract, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active(owner) != nil {
		return nil, errors.New("Bạn đang có một hợp đồng chưa xong.")
	}
	now := time.Now()
	k := &Contract{ID: "hd" + randHex(5), Owner: owner, OwnerName: ownerName, Roof: roof, Price: price, State: "open", Need: need, CreatedAt: now, Deadline: now.Add(ttl)}
	c.Contracts[k.ID] = k
	if err := c.saveLocked(); err != nil {
		delete(c.Contracts, k.ID)
		return nil, err
	}
	cp := *k
	return &cp, nil
}

func (c *ContractStore) List(me string) []Contract {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := []Contract{}
	for _, k := range c.Contracts {
		if k.State == "open" || k.Owner == me || k.Worker == me {
			if k.State == "open" || k.State == "taken" || time.Since(k.Deadline) < 24*time.Hour {
				out = append(out, *k)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (c *ContractStore) Take(id, worker, workerName string) (*Contract, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.Contracts[id]
	switch {
	case k == nil || k.State != "open":
		return nil, errors.New("Hợp đồng không còn nhận thợ.")
	case k.Owner == worker:
		return nil, errors.New("Không tự nhận hợp đồng của mình.")
	case time.Now().After(k.Deadline):
		return nil, errors.New("Hợp đồng đã quá hạn.")
	}
	for _, x := range c.Contracts {
		if x.Worker == worker && x.State == "taken" {
			return nil, errors.New("Bạn đang làm một hợp đồng khác.")
		}
	}
	k.Worker, k.WorkerName, k.State = worker, workerName, "taken"
	if err := c.saveLocked(); err != nil {
		k.Worker, k.WorkerName, k.State = "", "", "open"
		return nil, err
	}
	cp := *k
	return &cp, nil
}

// WorkerOf cho biết thợ có hợp đồng đang làm tại nhà chủ này (để được vào nhà thi công).
func (c *ContractStore) WorkerOf(owner, worker string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range c.Contracts {
		if k.Owner == owner && k.Worker == worker && k.State == "taken" {
			return true
		}
	}
	return false
}

// Work cộng một lần thi công; trả về hợp đồng sau khi cộng (State "done" nếu vừa hoàn thành).
// Thanh toán cho thợ được ghi vào hàng chờ cùng lúc đổi trạng thái — chỉ một lần.
func (c *ContractStore) Work(id, worker string, now time.Time) (*Contract, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.Contracts[id]
	switch {
	case k == nil || k.State != "taken" || k.Worker != worker:
		return nil, errors.New("Bạn không có hợp đồng này.")
	case now.After(k.Deadline):
		return nil, errors.New("Hợp đồng đã quá hạn.")
	case now.Sub(k.LastWork) < WorkGap:
		return nil, errors.New("Từ từ thôi, mỗi nhát búa cần một chút thời gian.")
	}
	k.Progress++
	k.LastWork = now
	if k.Progress >= k.Need {
		k.State = "done"
		c.creditLocked(worker, Credit{Amount: k.Price, Reason: "contract_pay", Ref: k.ID, Note: "Tiền công xây chuồng cho " + k.OwnerName})
	}
	if err := c.saveLocked(); err != nil {
		k.Progress--
		k.State = "taken"
		return nil, err
	}
	cp := *k
	return &cp, nil
}

// Cancel: chủ huỷ khi chưa có thợ, hoặc thợ bỏ việc; ký quỹ hoàn chủ một lần.
func (c *ContractStore) Cancel(id, by string) (*Contract, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := c.Contracts[id]
	switch {
	case k == nil:
		return nil, errors.New("Không tìm thấy hợp đồng.")
	case k.State == "open" && k.Owner == by, k.State == "taken" && k.Worker == by:
	case k.State == "taken" && k.Owner == by:
		return nil, errors.New("Thợ đã nhận việc: không huỷ ngang; quá hạn chưa xong sẽ tự hoàn tiền.")
	default:
		return nil, errors.New("Không huỷ được hợp đồng này.")
	}
	k.State = "cancelled"
	c.creditLocked(k.Owner, Credit{Amount: k.Price, Items: k.Items, Reason: "contract_refund", Ref: k.ID, Note: "Hoàn ký quỹ và vật liệu hợp đồng xây chuồng"})
	if err := c.saveLocked(); err != nil {
		return nil, err
	}
	cp := *k
	return &cp, nil
}

// Expire hoàn ký quỹ các hợp đồng quá hạn chưa xong; trả về chủ cần báo.
func (c *ContractStore) Expire(now time.Time) []*Contract {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []*Contract
	for _, k := range c.Contracts {
		if (k.State == "open" || k.State == "taken") && now.After(k.Deadline) {
			k.State = "expired"
			c.creditLocked(k.Owner, Credit{Amount: k.Price, Items: k.Items, Reason: "contract_refund", Ref: k.ID, Note: "Hợp đồng quá hạn, hoàn ký quỹ và vật liệu"})
			cp := *k
			out = append(out, &cp)
		}
	}
	if len(out) > 0 {
		_ = c.saveLocked()
	}
	return out
}

// applyCredits cộng các khoản chờ vào ví (gọi trong goroutine phòng giữ tài khoản).
func (r *Room) applyCredits(p *Player, now time.Time) {
	cs := r.hub.store.Contracts.TakeCredits(p.acc.ID)
	if len(cs) == 0 {
		return
	}
	snap, _ := json.Marshal(p.acc)
	for _, c := range cs {
		_ = p.acc.addThoc(c.Amount, c.Reason, c.Ref, now)
		for id, n := range c.Items {
			p.acc.Items[id] += n // hoàn vật liệu kể cả khi kho đã đầy: không để mất tài sản
		}
	}
	if err := r.hub.store.Save(p.acc); err != nil {
		restore(p.acc, snap)
		r.hub.store.Contracts.Restore(p.acc.ID, cs)
		return
	}
	for _, c := range cs {
		r.toast(p, "good", fmt.Sprintf("%s: +%d thóc", c.Note, c.Amount))
	}
	p.sess.Send(map[string]any{"t": "me", "me": p.acc.View(now, r.hub.bal)})
}

func weekKey(t time.Time) string {
	y, w := t.In(vnZone).ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// contractAct: tạo/nhận/thi công/huỷ hợp đồng.
func (r *Room) contractAct(p *Player, m *inMsg, reply func(ActResult), fail func(error), txn func(func() (ActResult, error))) {
	now := time.Now()
	b := r.hub.bal
	cs := r.hub.store.Contracts
	switch m.A {
	case "contract_create":
		h := r.home
		switch {
		case h == nil || h.Owner != p.acc.ID:
			fail(errors.New("Đăng hợp đồng trong chuồng riêng của bạn."))
			return
		case nearKind(r.zones["home"], p.pos, "npc_cow") == nil:
			fail(errors.New("Gặp Bò Mộc để đăng hợp đồng thuê thợ."))
			return
		case h.Tier < 1 || h.Coop != nil:
			fail(errors.New("Cần Mảnh đất riêng và chưa có chuồng để thuê thợ xây."))
			return
		case RoofColors[m.ID] == "":
			fail(errors.New("Chọn màu mái hợp lệ."))
			return
		case m.I < ContractMinPrice || m.I > ContractMaxPrice:
			fail(fmt.Errorf("Tiền công từ %d đến %d thóc.", ContractMinPrice, ContractMaxPrice))
			return
		}
		material := m.Mode
		if material == "" {
			material = "go"
		}
		rc, ok := HouseDesigns[0].Recipes[material]
		if !ok {
			fail(errors.New("Chọn vật liệu: rơm, gỗ hoặc gạch."))
			return
		}
		txn(func() (ActResult, error) {
			// Chủ phải gom đủ vật liệu: giữ cùng ký quỹ, thợ chỉ góp công (PLAN_LOGIC_GAME 13.3).
			if err := p.acc.takeItems(rc.Items); err != nil {
				return ActResult{}, err
			}
			if err := p.acc.addThoc(-m.I, "contract_escrow", "new", now); err != nil {
				return ActResult{}, err
			}
			k, err := cs.Create(p.acc.ID, p.acc.Char.Name, m.ID, m.I, b.ContractWork, b.ContractTTL)
			if err != nil {
				return ActResult{}, err
			}
			cs.SetMaterials(k.ID, material, rc.Items)
			return ActResult{Msg: fmt.Sprintf("Đã đăng hợp đồng: giữ %d thóc ký quỹ, hạn %s.", k.Price, humanDur(b.ContractTTL))}, nil
		})
	case "contract_take":
		if p.acc.Level < ContractLevel {
			fail(fmt.Errorf("Nghề thợ xây mở từ cấp %d.", ContractLevel))
			return
		}
		k, err := cs.Take(m.ID, p.acc.ID, p.acc.Char.Name)
		if err != nil {
			fail(err)
			return
		}
		owner := k.Owner
		go func() {
			if r.hub.store.Social.AddMail(owner, "build", "🔨 "+k.WorkerName+" đã nhận hợp đồng xây chuồng của bạn.") {
				r.hub.mailPing(owner)
			}
		}()
		reply(ActResult{OK: true, Msg: "Đã nhận việc. Vào chuồng của " + k.OwnerName + " và thi công ở nền chuồng.", Data: map[string]any{"owner": k.Owner}})
	case "contract_work":
		h := r.home
		if h == nil || p.zone != "home" {
			fail(errors.New("Thi công trong chuồng của chủ hợp đồng."))
			return
		}
		cx, cz := coopCell(h.tier())
		if p.pos.Dist(Vec{float64(cx), float64(cz) - 0.5}) > 3.2 {
			fail(errors.New("Đứng cạnh nền chuồng để thi công."))
			return
		}
		if h.Coop != nil {
			fail(errors.New("Chuồng đã có, không cần xây nữa."))
			return
		}
		k, err := cs.Work(m.ID, p.acc.ID, now)
		if err != nil || k.Owner != h.Owner {
			if err == nil {
				err = errors.New("Hợp đồng không thuộc chuồng này.")
			}
			fail(err)
			return
		}
		r.broadcastZone(p.zone, nil, map[string]any{"t": "fx", "pid": p.PID, "fx": "dig"})
		if k.State != "done" {
			reply(ActResult{OK: true, Msg: fmt.Sprintf("🔨 Tiến độ %d/%d", k.Progress, k.Need), Data: map[string]any{"progress": k.Progress, "need": k.Need}})
			return
		}
		// Nghiệm thu tự động: dựng chuồng theo mẫu, trả công qua hàng chờ (đã ghi trong Work).
		mat := k.Material
		if mat == "" {
			mat = "go"
		}
		h.Coop = &CoopJob{Roof: k.Roof, Design: "chuong_nho", Material: mat, State: "done", StartAt: now, DoneAt: now}
		if err := r.hub.store.Homes.Save(h); err != nil {
			h.Coop = nil
		}
		pair := k.Owner + ":" + weekKey(now)
		if !p.acc.RepPairs[pair] {
			p.acc.RepPairs[pair] = true
			p.acc.Reputation++
		}
		p.dirty = true
		r.broadcastHome()
		r.applyCredits(p, now)
		owner := k.Owner
		go func() {
			if r.hub.store.Social.AddMail(owner, "build", "🏠 "+k.WorkerName+" đã xây xong chuồng theo hợp đồng của bạn!") {
				r.hub.mailPing(owner)
			}
		}()
		reply(ActResult{OK: true, Msg: "🏠 Hoàn thành! Máy chủ đã nghiệm thu và trả công."})
	case "contract_cancel":
		k, err := cs.Cancel(m.ID, p.acc.ID)
		if err != nil {
			fail(err)
			return
		}
		r.applyCredits(p, now)
		reply(ActResult{OK: true, Msg: "Đã huỷ hợp đồng " + k.ID + ", ký quỹ hoàn về chủ."})
	}
}
