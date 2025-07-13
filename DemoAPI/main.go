package main

import (
	"fmt"
	"hello-go/v2/hello"
	"net/http"
)

func helloHandler(resp http.ResponseWriter, req *http.Request) {
	name := "Vast, cold world"
	if req.URL.Query().Has("name") {
		name = req.URL.Query().Get("name")
	}
	fmt.Fprintf(resp, hello.SayHello(name))
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	fmt.Println("Startup on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
	}
}
