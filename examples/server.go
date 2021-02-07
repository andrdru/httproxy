package main

import (
	"math/rand"
	"net/http"
	"os"
)

var port = "80"
var host = ""
var broken = ""

func main() {

	host = os.Getenv("HOST")
	var p = os.Getenv("SERVER_PORT")
	if p != "" {
		port = p
	}

	broken = os.Getenv("BROKEN")

	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/", HelloHandler)
	http.ListenAndServe(":"+port, nil)
}

func HealthHandler(writer http.ResponseWriter, request *http.Request) {
	if broken != "" && rand.Int()%4 != 0 {
		writer.WriteHeader(500)
	}
}

func HelloHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Write([]byte("hello from " + host + ":" + port))
}
