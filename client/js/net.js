// Kết nối WebSocket tới máy chủ Go. Client chỉ gửi input/ý định; mọi kết quả do server trả về.

export class Net {
  constructor() {
    this.handlers = {};
    this.pending = new Map(); // rid → { resolve, timer, msg }
    this.ws = null;
    this.token = null;
    this.retry = 0;
    this.stopped = false;
    this.rtt = 0;
    this.dropped = false; // true khi vừa mất mạng thật; main.js dùng để chỉ báo "đã kết nối lại" đúng lúc
  }

  on(type, fn) { this.handlers[type] = fn; }

  // resume=true khi tự nối lại sau mất mạng: server tiếp tục phiên cũ nếu còn trong thời gian giữ chỗ.
  connect(token, resume = false) {
    this.token = token;
    this.stopped = false;
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const ws = new WebSocket(`${proto}://${location.host}/ws`);
    this.ws = ws;
    ws.onopen = () => {
      this.retry = 0;
      this.raw({ t: 'hello', token, resume });
      this.handlers.open?.();
      // Gửi lại các hành động chưa có phản hồi với cùng request ID: server trả kết quả cũ, không xử lý lần hai.
      for (const p of this.pending.values()) this.raw(p.msg);
    };
    ws.onmessage = (ev) => {
      let m;
      try { m = JSON.parse(ev.data); } catch { return; }
      if (m.t === 'res' && m.rid && this.pending.has(m.rid)) {
        const p = this.pending.get(m.rid);
        clearTimeout(p.timer);
        this.pending.delete(m.rid);
        p.resolve(m);
      }
      if (m.t === 'pong') this.rtt = performance.now() - m.c;
      this.handlers[m.t]?.(m);
    };
    ws.onclose = () => {
      this.handlers.close?.();
      if (this.stopped) return;
      this.dropped = true;
      const wait = Math.min(8000, 500 * 2 ** this.retry++);
      setTimeout(() => !this.stopped && this.connect(this.token, true), wait);
    };
  }

  stop() { this.stopped = true; this.ws?.close(); }

  raw(msg) {
    if (this.ws && this.ws.readyState === 1) this.ws.send(JSON.stringify(msg));
  }

  // act gửi hành động có request ID và chờ kết quả từ server.
  act(a, args = {}) {
    const rid = (crypto.randomUUID ? crypto.randomUUID() : String(Math.random()).slice(2)) ;
    const msg = { t: 'act', rid, a, ...args };
    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        this.pending.delete(rid);
        resolve({ ok: false, err: 'Máy chủ chưa phản hồi. Kiểm tra mạng rồi thử lại.' });
      }, 10000);
      this.pending.set(rid, { resolve, timer, msg });
      this.raw(msg);
    });
  }

  ping() { this.raw({ t: 'ping', c: performance.now() }); }
}

export async function api(path, body) {
  try {
    const r = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    return await r.json();
  } catch {
    return { err: 'Không kết nối được máy chủ.' };
  }
}
