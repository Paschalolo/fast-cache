package main

import (
	"fmt"
	"net/http"
	"time"
)

func main() {
	cs := NewCacheServer()
	cs.cache.startEvictionTicker(time.Minute * 1)
	http.HandleFunc("/set", cs.SetHandler)
	http.HandleFunc("/get", cs.GetHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
		return
	}
}
