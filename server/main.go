package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	addr := flag.String("addr", ":8080", "địa chỉ lắng nghe")
	data := flag.String("data", "data", "thư mục lưu tài khoản và sổ cái")
	static := flag.String("static", "../client", "thư mục client (Three.js)")
	balance := flag.String("balance", "live", "bộ cân bằng: live (chơi thật, PLAN_LOGIC_GAME) | demo (rút ngắn để playtest) | plan (số trong PLAN.md)")
	staff := flag.String("staff", "", "tài khoản quản trị, vd: an:mod,binh:econ,chu:admin")
	devWarp := flag.Bool("devwarp", false, "cho phép nhảy thẳng tới khu (chỉ để chụp ảnh đối chiếu giao diện, không bật khi chơi thật)")
	flag.Parse()

	var bal Balance
	switch *balance {
	case "live":
		bal = LiveBalance()
	case "demo":
		bal = DemoBalance()
	case "plan":
		bal = PlanBalance()
	default:
		log.Fatalf("bộ cân bằng %q không có (live | demo | plan)", *balance)
	}
	store, err := OpenStore(*data)
	if err != nil {
		log.Fatalf("mở dữ liệu: %v", err)
	}
	if _, err := os.Stat(filepath.Join(*static, "index.html")); err != nil {
		log.Fatalf("không thấy client tại %s (dùng -static)", *static)
	}
	hub := NewHub(store, bal)
	hub.staff = ParseStaff(*staff)
	hub.devWarp = *devWarp

	mux := hub.Routes()
	files := http.FileServer(http.Dir(*static))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	log.Printf("Trang trại gà Hypex — http://localhost%s (cân bằng: %s)", *addr, bal.Name)
	log.Fatal(srv.ListenAndServe())
}
