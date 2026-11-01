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
	balance := flag.String("balance", "demo", "bộ cân bằng: demo (rút ngắn để playtest) | plan (số trong PLAN.md)")
	staff := flag.String("staff", "", "tài khoản quản trị, vd: an:mod,binh:econ,chu:admin")
	flag.Parse()

	bal := DemoBalance()
	if *balance == "plan" {
		bal = PlanBalance()
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
