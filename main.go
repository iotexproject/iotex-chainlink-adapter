package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("/adopt", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var dst map[string]interface{}
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&dst); err != nil {
			log.Println("decode error", err)
		}
		log.Printf("%+v", dst)
	}))
	http.ListenAndServe(":9079", mux)
}
