package main

import (
	"benchmark.local/checkout/app"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	log.Fatal(http.ListenAndServe(addr, http.HandlerFunc(app.Handler)))
}
