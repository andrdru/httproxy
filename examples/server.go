package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"
)

var port = "80"
var host = ""
var broken = ""

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	var ctx, cancel = context.WithCancel(context.Background())

	go func() {
		oscall := <-c
		log.Printf("system call: %+v", oscall)
		cancel()
	}()

	host = os.Getenv("HOST")
	var p = os.Getenv("SERVER_PORT")
	if p != "" {
		port = p
	}

	broken = os.Getenv("BROKEN")

	var mux = http.NewServeMux()

	mux.Handle("/health", http.HandlerFunc(HealthHandler))
	mux.HandleFunc("/", HelloHandler)

	var srv = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	var err error
	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("started")

	<-ctx.Done()

	log.Printf("stopping")

	var ctxCancel context.Context
	ctxCancel, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = srv.Shutdown(ctxCancel); err != nil {
		log.Fatalf("stop failed: %s", err)
	}

	log.Printf("stopped")
}

func HealthHandler(writer http.ResponseWriter, request *http.Request) {
	if broken != "" && rand.Int()%4 != 0 {
		writer.WriteHeader(500)
	}
}

func HelloHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("hello from " + host + ":" + port))
}
