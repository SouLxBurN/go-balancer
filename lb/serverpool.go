package lb

import (
	"container/heap"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

// boolMap Converter map of true false values
// to string representations of up and down.
var boolMap = map[bool]string{true: "up", false: "down"}

// ServerPool Heap interface
// contains priorty queue of ServerNode.
type ServerPool struct {
	queue []*ServerNode
	mux   sync.Mutex
}

// NewPool Instatiates a new ServerPool
// with an empty queue.
func NewPool() *ServerPool {
	return &ServerPool{
		queue: []*ServerNode{},
	}
}

// Len sort.Interface length method.
func (s *ServerPool) Len() int {
	return len(s.queue)
}

// Less sort.Interface comparison method.
func (s *ServerPool) Less(i, j int) bool {
	return len(s.queue[i].ActiveRequests) < len(s.queue[j].ActiveRequests)
}

// Swap sort.Interface it swaps.
func (s *ServerPool) Swap(i, j int) {
	s.queue[i], s.queue[j] = s.queue[j], s.queue[i]
	s.queue[i].poolIndex = i
	s.queue[j].poolIndex = j
}

// Push heap.Interface adds ServerNode to the end of the queue
func (s *ServerPool) Push(x interface{}) {
	length := len(s.queue)
	node := x.(*ServerNode)
	node.poolIndex = length
	s.queue = append(s.queue, node)
}

// Pop heap.Interface removes and returns value at the
// end of the ServerNode queue
func (s *ServerPool) Pop() interface{} {
	end := len(s.queue) - 1
	node := s.queue[end]
	s.queue[end] = nil
	s.queue = s.queue[:end]
	node.poolIndex = -1

	return node
}

// AddRequestToNode Creates a uuid for a http.request and attaches it
// to a ServerNode for tracking the number of requests pending on a node.
// Additionally removes the request when the request's context is Done().
func (s *ServerPool) AddRequestToNode(node *ServerNode, req *http.Request) {
	s.mux.Lock()
	defer s.mux.Unlock()
	uuid := uuid.New().String()
	node.AddActiveRequest(uuid, req)
	heap.Fix(s, node.poolIndex)

	go func() {
		for {
			select {
			case <-req.Context().Done():
				s.mux.Lock()
				defer s.mux.Unlock()
				node.RemoveRequest(uuid)
				heap.Fix(s, node.poolIndex)
				return
			}
		}
	}()
}

// GetNextNode returns the next ServerNode available
// for traffic
func (s *ServerPool) GetNextNode() *ServerNode {
	s.mux.Lock()
	defer s.mux.Unlock()
	length := s.Len()
	if length < 1 {
		return nil
	}

	next := heap.Pop(s).(*ServerNode)

	for next != nil && !next.IsAlive() {
		if s.Len() >= 1 {
			next = heap.Pop(s).(*ServerNode)
		} else {
			log.Println("No healthy hosts")
			return nil
		}
	}

	heap.Push(s, next)
	return next
}

// RegisterNode registers a new node to the server pool
func (s *ServerPool) RegisterNode(node *ServerNode) {
	s.mux.Lock()
	defer s.mux.Unlock()
	heap.Push(s, node)
}

// DeregisterNode removes a node from the ServerPool
// based matching on URL
func (s *ServerPool) DeregisterNode(nodeURL string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	for _, node := range s.queue {
		if node != nil && node.URL.String() == nodeURL {
			heap.Remove(s, node.poolIndex)
		}
	}
}

// HealthChecks runs health checks on all nodes in
// the ServerPool
func (s *ServerPool) HealthChecks(t *time.Ticker) {
	for {
		select {
		case <-t.C:
			log.Println("Running Health Checks...")
			for _, b := range s.queue {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
				log.Printf("%s [%s] - %d\n", b.URL, boolMap[alive], len(b.ActiveRequests))
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
