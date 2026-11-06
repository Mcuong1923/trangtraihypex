# Trang trại gà Hypex

Thế giới nông trại 3D nhiều người chơi trên trình duyệt, làm theo [PLAN.md](PLAN.md) và bộ ảnh trong [assets/illustrations/](assets/illustrations/).

**Plan làm lại giao diện theo bộ ảnh 2 (02/11/2026): [PLAN_GIAO_DIEN_BO_2.md](PLAN_GIAO_DIEN_BO_2.md).** Đặc tả camera, HUD gỗ–kem, kiến trúc/vật liệu và bố cục đủ 11 cảnh, danh sách tài nguyên, trình tự triển khai và đối chiếu ảnh. Đây là kế hoạch mới, chưa phải báo cáo đã triển khai; bộ 2 được ưu tiên khi có khác biệt về mỹ thuật.

**Thế giới 3D mới (01/11/2026), theo [PLAN_THE_GIOI_3D.md](PLAN_THE_GIOI_3D.md):** sảnh chuồng cộng đồng 40 ổ có Cụ Cục ngồi trên bục và NPC sinh hoạt; camera kéo xoay ngang/đứng, zoom, chạm đất để đi; ra cửa sảnh là một bản đồ lớn liền mạch, đi bộ theo đường làng (không dịch chuyển) tới vườn giun, chợ, sân PK, trạm ấp, ao, Xóm Rơm Vàng (nhà riêng, Nhà Hội) và bến xe; bản đồ làng/các trang trại có “Đặt chỉ đường”.

Máy chủ Go quyết định toàn bộ luật chơi (vị trí, đào giun, thóc, mana, giao dịch, PK). Client Three.js chỉ gửi input và hiển thị.

**Trạng thái từng hạng mục (đã xong / server xong nhưng chưa có giao diện / chưa làm): xem [TIEN_DO.md](TIEN_DO.md).**

## Chạy

Cần Go 1.24 trở lên.

```bash
cd server
go run .                                   # http://localhost:8080, bộ cân bằng "live" (chơi thật, PLAN_LOGIC_GAME)
go run . -balance demo                     # rút ngắn mọi thời gian để thử nhanh
go run . -balance plan                     # dùng đúng số gốc trong PLAN.md để đối chiếu
go run . -staff an:mod,binh:econ,chu:admin # cấp quyền quản trị cho các tài khoản này
go run . -devwarp                          # chỉ để chụp ảnh đối chiếu giao diện: cho phép nhảy thẳng tới khu
```

- **Chơi:** mở `http://localhost:8080`, tạo tài khoản rồi tạo gà. Để thử nhiều người chơi, mở thêm cửa sổ ẩn danh với tài khoản thứ hai.
- **Quản trị:** mở `http://localhost:8080/admin.html` và đăng nhập bằng tài khoản có trong `-staff`. Vai trò `mod` kiểm duyệt; `econ` xem sổ cái; `admin` làm được tất cả, kể cả bật bảo trì và xem nhật ký quản trị.
- Client tải Three.js r128 và font từ CDN, nên trình duyệt cần có mạng.

### Ba bộ cân bằng

| Thông số | `live` (mặc định) | `demo` (thử nhanh) | `plan` (số gốc PLAN.md) |
| --- | --- | --- | --- |
| Hồi đất giữa 2 lượt đào | 8 giây (mưa −30%) | 4 giây | 45 giây |
| Độ no giảm khi online | 4,5 điểm/phút (đầy bụng ~15–20 phút) + 0,4/lượt đào | như `live` | 1 điểm/phút + 1/lượt đào |
| Đói kéo dài / kiệt sức | 3 / 6 phút ở độ no 0 | 3 / 6 phút | 5 / 10 phút |
| Quà khởi nghiệp · thưởng hướng dẫn | 100 thóc · 50 thóc + 3 cám + 1 trứng | như `live` | 100 thóc · 100 thóc |
| Bò Mộc xây nhà | 10–60 phút theo thiết kế, vật liệu | ×1/30 (20 giây cho mỗi 10 phút) | 10 phút |
| Ấp trứng | 30 phút | 30 giây | 2 giờ |
| Xe liên trại xuất bến | mỗi 3 phút | mỗi 90 giây | — |
| Cây lớn | 10–45 phút theo hạt | ×1/10 | 10–45 phút |

Lượt xe miễn phí (tối đa 2, hồi 1 lượt/5 phút), giá vé, xác suất giun, giằng co, XP và các giới hạn khác giống nhau ở mọi bộ. Luật chi tiết: [PLAN_LOGIC_GAME.md](PLAN_LOGIC_GAME.md).

## Kiểm thử

```bash
cd server
go test -count=1 -timeout 25m ./...                              # 122 test: luật + end-to-end qua WebSocket (~12 phút); -race cần cgo/gcc
NODE=/duong/dan/node go test -run TestTugMatchesClient .         # so khớp giằng co Go ↔ client/js/tug.js (bỏ qua nếu không có Node)
go run ./cmd/loadtest -url http://localhost:8080 -n 1000 -dur 3m # load test với bot
```

Load test mô phỏng tỉ lệ hành vi ở mục 20: 60% đi lại/tương tác, 15% về nhà, 15% PK, 10% chat. Kết quả đo gần nhất nằm trong [TIEN_DO.md](TIEN_DO.md).

## Kiến trúc

```
Trình duyệt (Three.js)  ──WebSocket: input, ý định──▶  Máy chủ Go
   hiển thị, camera, UI   ◀──snapshot 10 lần/giây, kết quả──   mô phỏng 20 tick/giây, luật, lưu trữ
```

| File (server/) | Vai trò |
| --- | --- |
| `config.go`, `world.go` | Bảng giun/thức ăn/thời trang/xe, cân bằng; bố cục các khu và vùng (trại Hypex, trại Ven Suối) |
| `account.go`, `community.go`, `v11.go` | Luật dạng hàm thuần: đào, ăn, mana, nhiệm vụ, danh hiệu, ấp trứng, vé xe, mua xe |
| `room.go`, `actions.go`, `v11_room.go` | Vòng lặp phòng 20 tick/giây, di chuyển do server tính, giao dịch idempotent, xe buýt, xe chở bạn |
| `pk.go`, `trade.go`, `contracts.go`, `guild.go` | PK, giao dịch hai bên, hợp đồng thợ xây, Hội Gà |
| `home.go`, `home_actions.go` | Nhà riêng: đất, Bò Mộc, đặt đồ, quyền vào |
| `social.go`, `social_hub.go` | Bạn bè, tin riêng, chặn, báo cáo, thư Chim Chíp, thăm nhà |
| `hub.go`, `store.go`, `admin.go` | Phiên WebSocket và nối lại, xếp phòng theo vùng, đăng nhập/khôi phục, lưu trữ, quản trị và số đo |
| `dig.go`, `fishing.go`, `garden.go`, `items.go`, `daily.go`, `pets.go`, `estate.go`, `showcase.go`, `logic_actions.go` | PLAN_LOGIC_GAME: giằng co giun, câu cá, trồng cây, kho đồ, quà cấp và điểm danh, kỹ năng thú cưng, mua đất và xây nhà, lô trưng bày ở Xóm |
| `npc.go`, `npc_director.go`, `npc_lines.go`, `roads.go`, `transit.go` | Làng sống: xe buýt vòng làng, xe ôm, giao thông NPC, cameo, trận biểu diễn, lời thoại, hướng dẫn qua NPC |
| `clock.go`, `weekly.go` | Ngày/đêm, mưa; bảng xếp hạng tuần |
| `cmd/loadtest/` | Công cụ load test |

Client (`client/js/`):
- **Cảnh 3D:** `world.js` dựng cảnh, `models.js` chứa mô hình gà/NPC/xe/đồ trang trí.
- **Giao diện:** `panels.js`, `social.js`, `v11.js`, `life.js` (đất, nhà, vật liệu, trồng cây, cài đặt NPC) chứa các bảng; `admin.js` là trang quản trị.
- **Làng sống:** `npc.js` (xe, NPC, bong bóng, xe ôm, bến xe, trận biểu diễn), `dig.js` + `tug.js` (đào giun giằng co; `tug.js` là bản sao 1-1 mô phỏng của máy chủ), `fish.js` (câu cá ở ao).
- **Điều phối:** `main.js`.

Nguyên tắc từ mục 16 của plan được giữ:
- Client không gửi toạ độ hay kết quả.
- Mọi thay đổi tài sản có request ID, được lưu bền vững trước khi báo thành công, và hoàn nguyên nếu ghi lỗi.
- Mỗi tài khoản chỉ một phiên điều khiển. Mỗi phiên mới bắt đầu ở sảnh chung; mất mạng thì có 15 giây để nối lại.

## Khác plan

- **Máy chủ:** Go + gorilla/websocket, theo kiến trúc chủ dự án chọn, thay cho Node.js + Colyseus.
- **Lưu trữ:** file JSON ghi nguyên tử kèm sổ cái `ledger.jsonl`, thay cho PostgreSQL + Redis. Giao diện `Store` đã tách riêng để thay sau.
- **Số tiến trình:** mới chạy một tiến trình (plan yêu cầu ít nhất 2 instance).
