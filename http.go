package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	spewg "github.com/Paschalolo/fast-cache/hash"
)

const replicationHeader = "X-Replication-Request"

type CacheServer struct {
	cache    *Cache
	peers    []string
	SelfID   string
	hashRing *spewg.HashRing
	sync.Mutex
}

func NewCacheServer(capacity int, peers []string, selfID string) *CacheServer {
	cs := &CacheServer{
		cache:    NewCache(capacity),
		peers:    peers,
		SelfID:   selfID,
		hashRing: spewg.NewHashRing(),
	}
	for _, peer := range peers {
		cs.hashRing.AddNode(spewg.Node{ID: peer, Addr: peer})
	}
	cs.hashRing.AddNode(spewg.Node{ID: selfID, Addr: "self"})
	return cs
}

func (cs *CacheServer) replicaset(key, value string, ttl time.Duration) {
	cs.Lock()
	defer cs.Unlock()

	req := struct {
		Key   string        `json:"key"`
		Value string        `json:"value"`
		Ttl   time.Duration `json:"ttl"`
	}{
		Key:   key,
		Value: value,
		Ttl:   ttl,
	}
	data, err := json.Marshal(req)
	if err != nil {
		log.Fatal("cannot json marshal")
		return
	}
	for _, peer := range cs.peers {
		if peer != cs.SelfID {
			go func(peer string) {
				client := http.Client{}
				req, err := http.NewRequest("POST", peer+"/set", bytes.NewReader(data))
				if err != nil {
					log.Fatal("cannot json marshal")
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set(replicationHeader, "true")
				if _, err = client.Do(req); err != nil {
					log.Printf("failed to replicate peer %s : %v", peer, err)
					return
				}
				log.Println("Replication succesfult to ", peer)
			}(peer)
		}
	}

}
func (cs *CacheServer) SetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string        `json:"key"`
		Value string        `json:"value"`
		Ttl   time.Duration `json:"ttl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	targetNode := cs.hashRing.GetNode(req.Key)
	if targetNode.Addr == "self" {
		cs.cache.Set(req.Key, req.Value, req.Ttl)
		if r.Header.Get(replicationHeader) == "" {
			go cs.replicaset(req.Key, req.Value, req.Ttl)
		}
		w.WriteHeader(http.StatusOK)
	} else {
		cs.forwardRequest(&targetNode, r)
	}
}

func (cs *CacheServer) forwardRequest(targetNode *spewg.Node, r *http.Request) {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 100, // Adjust based on your load
		},
		Timeout: 5 * time.Second, // Prevent requests from hanging indefinitely
	}

	// Create a new request based on the method
	var req *http.Request
	var err error

	if r.Method == http.MethodGet {
		// Forward GET request with query parameters
		getURL := fmt.Sprintf("%s%s?%s", targetNode.Addr, r.URL.Path, r.URL.RawQuery)
		req, err = http.NewRequest(r.Method, getURL, nil)
	} else if r.Method == http.MethodPost {
		// Forward POST request with body
		postURL := fmt.Sprintf("%s%s", targetNode.Addr, r.URL.Path)
		req, err = http.NewRequest(r.Method, postURL, r.Body)
	}

	if err != nil {
		log.Printf("Failed to create forward request: %v", err)
		return
	}

	// Copy the headers
	req.Header = r.Header

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		// Check for a "connection refused" error
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Err != nil {
			var opErr *net.OpError
			if errors.As(urlErr.Err, &opErr) && opErr.Op == "dial" {
				var sysErr *os.SyscallError
				if errors.As(opErr.Err, &sysErr) && sysErr.Syscall == "connect" {
					log.Printf("Connection refused to node %s: %v", targetNode.Addr, err)
					// Consider adding retry logic or node status checks here
					return
				}
			}
		}
		log.Printf("Failed to forward request to node %s: %v", targetNode.Addr, err)
		return
	}
	io.ReadAll(resp.Body)
}
func (cs *CacheServer) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	targetNode := cs.hashRing.GetNode(key)
	if targetNode.Addr == "self" {
		value, found := cs.cache.Get(key)

		if !found {
			http.NotFound(w, r)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]string{"value": value}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		originalSender := r.Header.Get("X-Forwarded-For")
		if originalSender == cs.SelfID {
			http.Error(w, "loop detected ", http.StatusInternalServerError)
			return
		}
		r.Header.Set("X-Forwarded-For", cs.SelfID)
		cs.forwardRequest(&targetNode, r)
	}

}
