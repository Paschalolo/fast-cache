package main

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

var (
	port  string
	peers string
)

func main() {
	flag.StringVar(&port, "port", ":8080", "HTTP server port")
	flag.StringVar(&peers, "peers", "", "comma seperated list of peers")
	flag.Parse()
	peerList := strings.Split(peers, ",")
	nodeID := fmt.Sprintf("%s%d", "node", rand.IntN(100))
	cs := NewCacheServer(100, peerList, nodeID)
	cs.cache.startEvictionTicker(time.Minute * 1)
	http.HandleFunc("/set", cs.SetHandler)
	http.HandleFunc("/get", cs.GetHandler)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Println(err)
		return
	}
}
