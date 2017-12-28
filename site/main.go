package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request GET %s\n", r.URL.String())
	log.Printf("Headers: %+v\n", r.Header)

	fmt.Fprintf(w, "Hi user %s!\n\n", r.Header["X-Auth-User-Id"][0])
	fmt.Fprintf(w, "Request GET %s\nHeaders:\n", r.URL.String())
	for k, v := range r.Header {
		fmt.Fprintf(w, " - %s: %s\n", k, v)
	}

}

func main() {
	log.SetOutput(os.Stdout)

	http.HandleFunc("/", handler)
	http.ListenAndServe(":80", nil)
}
