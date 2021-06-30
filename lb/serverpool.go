package lb

import (
	"log"
	"net"
	"net/url"
	"sync/atomic"
	"time"
)

var boolMap = map[bool]string{true: "up", false: "down"}

type ServerPool struct {
	nodes   []*ServerNode
	current uint64
}

func (s *ServerPool) MarkBackendStatus(url *url.URL, alive bool) {
	for _, b := range s.nodes {
		if b.URL == url {
			b.SetAlive(alive)
			return
		}
	}
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.nodes)))
}

func (s *ServerPool) GetNextPeer() *ServerNode {
	if len(s.nodes) < 1 {
		log.Println("No registered hosts")
		return nil
	}
	next := s.NextIndex()
	l := len(s.nodes) + next
	for i := next; i < l; i++ {
		index := i % len(s.nodes)

		if s.nodes[index].Alive {
			if i != next {
				atomic.AddUint64(&s.current, uint64(index))
			}
			return s.nodes[index]
		}
	}
	return nil
}

// RegisterNode registers a new node to the server pool
// TODO: This is not thread safe.
func (s *ServerPool) RegisterNode(node *ServerNode) {
	s.nodes = append(s.nodes, node)
}

// HealthChecks runs health checks on all nodes in
// the ServerPool
func (s *ServerPool) HealthChecks() {
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.C:
			log.Println("Running Health Checks...")
			for _, b := range s.nodes {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
				log.Printf("%s [%s]\n", b.URL, boolMap[alive])
			}
			log.Println("Health Checks Completed.")
		}
	}
}

// isBackendAlive checks whethert a back in Alive
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
