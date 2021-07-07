package lb

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/google/uuid"
)

type ServerNode struct {
	URL            *url.URL
	Alive          bool
	mux            sync.RWMutex
	ActiveRequests map[string]*http.Request
	ReverseProxy   *httputil.ReverseProxy
}

func (n *ServerNode) AddActiveRequest(req *http.Request) {
	n.mux.Lock()
	if n.ActiveRequests == nil {
		n.ActiveRequests = make(map[string]*http.Request)
	}
	uuid := uuid.New().String()
	n.ActiveRequests[uuid] = req
	n.mux.Unlock()

	go func() {
		for {
			select {
			case <-req.Context().Done():
				n.mux.Lock()
				delete(n.ActiveRequests, uuid)
				n.mux.Unlock()
				return
			}
		}
	}()
}

func (n *ServerNode) SetAlive(alive bool) {
	n.mux.Lock()
	n.Alive = alive
	n.mux.Unlock()
}

func (n *ServerNode) IsAlive() bool {
	n.mux.RLock()
	alive := n.Alive
	n.mux.RUnlock()
	return alive
}
