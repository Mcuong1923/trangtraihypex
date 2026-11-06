# Code review: PK arena scoreboard + KO fx (feat/bang-diem-pk, uncommitted)

## Scope
- server/pk_scoreboard.go (new), server/pk.go hooks, server/room.go (field, welcome, tick), server/pk_scoreboard_test.go
- client/js/arena-scoreboard.js (new), client/js/npc.js (exhibition score), client/js/main.js (pk_board, cheer, koStars)
- Bash tool broken in this session (shell init error), so I could not run `git diff` or `go vet`. I reviewed the full files with Read/Grep instead.

## Overall
The server design is solid. All state changes happen on the Room goroutine (pkAct, stepPK, tick, sendWelcome, and the test's DoWait). `finishMatch` is the only place that deletes from `r.matches` and it always calls `pkBoardEnd`, so `pkb.id` cannot point at a missing match. Coalescing works: at most 5 msgs/s, and only the featured match is broadcast. `Session.Send` marshals synchronously, so sharing `pkb.last` is safe. Arena zone groups are built once (world.js:82 `continue`), so the board is not rebuilt and leaks no texture/geometry. Two real bugs, both client-visible.

## High

### H1. Client board goes stale after a room/channel switch or reconnect when the server board is idle
- pk_scoreboard.go:100-108: `sendPKBoard` sends nothing when there is no featured match and `last` is nil or idle.
- arena-scoreboard.js:9-17: the module-level `real` is never reset on `welcome`.
- Scenario: the player watches a match in channel 1, then switches channel, goes home, or disconnects during the 5s hold. The new or rejoined room is idle, so nothing is sent. The board stays stuck forever on an old "play" or "end" state, and for a "play" state the countdown keeps running from the stale `realEnds`.
- The same thing happens when the idle broadcast is missed while the player is in a home room. Zone groups persist across rooms, so the stale board shows again on return.
- Fix (server, covers reconnect too): add a final fallback in `sendPKBoard`:
  ```go
  if r.pkb.last != nil && r.pkb.last["state"] != "idle" { p.sess.Send(r.pkb.last); return }
  if r.world != nil { p.sess.Send(map[string]any{"t": "pk_board", "state": "idle"}) }
  ```
  Alternatively, or as well, call `SB.onBoard({ state: 'idle' })` at the top of the client `welcome` handler (main.js ~258).
- Add a test: c sees a match end, c moves to another room/channel and back, and c must get an idle `pk_board`.

### H2. The exhibition board often shows the declared winner with an equal or lower score
- The server picks `Winner` independently of `Seed` (npc_director.go:547). The client builds `hits` from the seed with a 50/50 `who` (npc.js:342), and `showBoard` (npc.js:353-358) counts those hits.
- With about 30 hits, the winner has a strictly higher count less than half the time. So more than half of the shows display "KO! X thắng" while X has fewer hits or the same number.
- Fix: in `onScene`, right after building `hits`, make the result consistent. This is deterministic, so every client still matches:
  ```js
  const w = m.winner; let d = 0; for (const h of hits) d += h.who === w ? 1 : -1;
  for (let i = hits.length - 1; d <= 0 && i >= 0; i--) if (hits[i].who !== w) { hits[i].who = w; d += 2; }
  ```
  This also makes the animation, where the winner lands the last blows, match the board.

## Medium

### M1. Test coverage gaps (pk_scoreboard_test.go)
The single test covers only the happy path plus a forfeit. Missing cases:
- (a) A second match starts while the first is featured. The board must switch to the newer match, and the older match ending must not change the board.
- (b) The hold expires while another match is still active. The board must switch to that match, not go idle.
- (c) A cancel during countdown must give `cancelled: true` and `winner: -1`.
- (d) A bot match: names must be [name, "Gà Máy"] and the bot's score must reach spectators.
- (e) The H1 regression.

### M2. A forfeit is shown as "KO!"
- arena-scoreboard.js:61 and :95: any `winner >= 0` draws "KO!" plus "X thắng". That includes a win because the opponent left, and coin-mode wins.
- The server already computes a reason. Fix: send `forfeit: leaver != "" && !cancelled` in the `pk_board` end message (pkBoardEnd needs the leaver flag). The client can then show "Đối thủ bỏ trận" and skip the KO stamp. Also consider no "KO!" for coin mode.

### M3. Defensive gap in stepPKBoard (pk_scoreboard.go:69-76)
- If `b.id != ""` but `r.matches[b.id] == nil`, the function returns forever and the board freezes. This cannot happen today because only `finishMatch` deletes, but it would break silently after a future refactor.
- Fix: in that branch, `if mt == nil { b.id, b.holdUntil = "", now }` so it falls through to the next-match/idle logic.

## Low
- pk_scoreboard.go:33: `left` is sent as a full-precision float (e.g. 2.999998712). Round it to 0.1s with `math.Round(left*10)/10`. Also, `modeName` and `rounds` are constants the client already has. This only trims bytes from a ≤5/s room-wide broadcast, so it is optional.
- Network volume is fine. Each featured match sends about 5 msgs/s × room size × about 200 B during peck play, and nothing extra for non-featured matches. Fighters get both `pk_score` and `pk_board`, which is acceptable.
- Pre-existing pattern: main.js:792-805 `koStars` creates 5 new `OctahedronGeometry` per KO. The effects loop (main.js:980) removes the object but never calls `geometry.dispose()`. `stars` and `confetti` do the same, so this is not a regression. Over a long session, dispose the geometry in the effect cleanup or reuse one shared geometry.
- arena-scoreboard.js:46: the rebuild path removes the old group without disposing its geometries, materials, or CanvasTexture. Zones are never rebuilt today, so this path never runs. Either delete the rebuild handling (YAGNI) or dispose properly.
- npc.js:357: `setExhibition` allocates an object every frame and `update()` calls `JSON.stringify` on it every frame. The cost is negligible, but you could compare `a`, `b` and `ko` before calling.

## Edge cases checked, OK
- A match ends while not featured: no-op (pkBoardEnd id guard).
- A cancelled match: shows "Trận bị huỷ" and is held for 5s.
- A newer match starts during the hold: it replaces the held result.
- Bot match: `names()` uses "Gà Máy" and the fake B player is not read.
- Reconnect (rebind → sendWelcome → sendPKBoard) gets the live board with recomputed `left`. The idle case is covered in H1.
- Rooms without a world: home and guild rooms have no arena, so no match and no board. Guild rooms keep `world` non-nil but have no arena zone, so the same applies.
- Late joiner during the hold gets the end result. Coin mode `round + 1` is clamped by `rounds`.
- Concurrency: no off-goroutine access. The test mutates `pkb` inside `DoWait`.

## Recommended actions
1. Fix H1 on the server: always send idle on welcome in world rooms. Add the regression test.
2. Fix H2: make the exhibition hits consistent with `winner` in `onScene`.
3. M2 forfeit display. M3 defensive fallthrough.
4. Add the M1 tests (multi-match switch, cancel, bot).

## Unresolved questions
- Is a newer match always taking the board (so an older match's finish is never shown) intended when several matches overlap?
- Should a coin-mode win show "KO!" at all?
- I could not run `go vet` or the test (Bash unavailable). The lead says the test passes.

## Đã sửa (06/11/2026)

- H1: client xoá bảng khi nhận `welcome` (đổi kênh/về nhà/nối lại); phòng mới gửi lại `pk_board` ngay sau welcome nếu có trận.
- H2: trận biểu diễn chỉnh lịch cú mổ từ seed để bên thắng luôn mổ nhiều hơn; mọi client đổi giống nhau.
- M1: thêm `TestPKBoardOverlappingMatchesAndCancel` (trận mới chiếm bảng, trận cũ xong không đổi bảng, hết giữ chuyển sang trận còn đấu, huỷ lúc đếm ngược).
- M2: gói kết thúc có `forfeit`; bảng ghi "đối thủ bỏ trận" thay cho "KO!". Úp ngửa vẫn hiện KO (gà thua vẫn ngã KO như Mổ nhanh).
- M3: `stepPKBoard` tự bỏ trận không còn tồn tại rồi chuyển trận khác/về trống.
- Low: `left` làm tròn 0,1 giây; bỏ nhánh dựng lại bảng (nhóm Sân tỷ thí chỉ dựng một lần). Hình sao KO không giải phóng: giữ như `stars`/`confetti` hiện có.
- Câu hỏi trận chồng nhau: giữ "trận mới nhất lên bảng" (đơn giản, khán giả xem trận đang nóng); kết quả trận cũ vẫn có trong chat hệ thống và hiệu ứng thắng/KO.
