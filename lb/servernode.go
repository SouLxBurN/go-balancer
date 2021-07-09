package lb

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type ServerNode struct {
	URL            *url.URL
	Alive          bool
	mux            sync.RWMutex
	ActiveRequests map[string]*http.Request
	ReverseProxy   *httputil.ReverseProxy
	poolIndex      int
}

func (n *ServerNode) AddActiveRequest(uuid string, req *http.Request) {
	n.mux.Lock()
	defer n.mux.Unlock()
	if n.ActiveRequests == nil {
		n.ActiveRequests = make(map[string]*http.Request)
	}
	n.ActiveRequests[uuid] = req
}

func (n *ServerNode) RemoveRequest(uuid string) {
	n.mux.Lock()
	defer n.mux.Unlock()
	if n.ActiveRequests == nil {
		n.ActiveRequests = make(map[string]*http.Request)
	}
	delete(n.ActiveRequests, uuid)
}

func (n *ServerNode) SetAlive(alive bool) {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.Alive = alive
}

func (n *ServerNode) IsAlive() bool {
	n.mux.RLock()
	defer n.mux.RUnlock()
	alive := n.Alive
	return alive
}
