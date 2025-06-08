# A DISTRUBUTED CACHE FOR APPLICATIONS 

Consistent Hasing and Peer to peer read replication is used to make this scalable .

To run two instances of the cache try 
```go 
    go run *.go -port=:8080 -peers=http://localhost:8081
    go run *.go -port=:8081 -peers=http://localhost:8080
```

## Todo list 
- cache replacement algorithms such as Low inter reference Recency set (LIRS).i.e can improve hit rates and better adapability
- Enable statefulSet on kubernetes for a distrubuted architecture using multi node failover 

- Compression : implement data compression techniques in order to store more data in memory and reduce memory footprint and potentially reduce payload size and hit rate 

- Eviction Policy : Tuning the eviction policy(LRU) based on specific data characteristics 

- connection pooling : optimize network commucnation by establishing network pooling between client and servers
- Implementing a leader follower consensus algorithm 
