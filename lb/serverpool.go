package lb

import (
	"log"
	"net"
	"net/url"
	"sync"
	"time"
)

var boolMap = map[bool]string{true: "up", false: "down"}

type ServerPool struct {
	queue []*ServerNode
	mux   sync.Mutex
}

// GetNextNode returns the next ServerNode available
// for traffic
func (s *ServerPool) GetNextNode() *ServerNode {
	s.mux.Lock()
	if len(s.queue) < 1 {
		log.Println("No registered hosts")
		return nil
	}

	var next *ServerNode
	for next = s.queue[0]; next != nil && !next.Alive; next = s.queue[0] {
		s.queue = s.queue[1:] // remove dead nodes from queue
	}
	if next == nil {
		log.Println("No healthy hosts")
		return nil
	}

	s.queue = s.queue[1:]           // dequeue
	s.queue = append(s.queue, next) // requeue in back of the line

	s.mux.Unlock()
	return next
}

// RegisterNode registers a new node to the server pool
func (s *ServerPool) RegisterNode(node *ServerNode) {
	s.mux.Lock()
	s.queue = append(s.queue, node)
	s.mux.Unlock()
}

// DeregisterNode removes a node from the ServerPool
// based matching on URL
func (s *ServerPool) DeregisterNode(nodeURL string) {
	s.mux.Lock()
	for i, v := range s.queue {
		if v.URL.String() == nodeURL {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
		}
	}
	s.mux.Unlock()
}

// HealthChecks runs health checks on all nodes in
// the ServerPool
func (s *ServerPool) HealthChecks() {
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.C:
			log.Println("Running Health Checks...")
			for _, b := range s.queue {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
				log.Printf("%s [%s]\n", b.URL, boolMap[alive])
			}
			log.Println("Health Checks Completed.")
		}
	}
}

// isBackendAlive checks whether a backend is Alive
// attempts to establish a http connection to verify liveness.
func isBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Node unreachable error: ", err)
		return false
	}

	_ = conn.Close()
	return true
}
