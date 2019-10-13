package server

import (
	"fmt"
	"net/http"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello World")
}

func start() {
	http.HandleFunc("/", helloWorld)
	http.ListenAndServe(":8080", nil)
}
