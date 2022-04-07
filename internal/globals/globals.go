package globals

import (
	"sync"
	"time"
)

// BackendSrv stores information for internal decision making
type BackendSrv struct {
	RW       sync.RWMutex
	Ip       string
	Reqs     int64
	Wt       float64
	RcvTime  time.Time
	LastRTT  uint64
	WtAvgRTT float64
	NoSched  bool
	Count    uint64
	Start    uint64
	End      uint64
	Grp      bool
}

// Endpoints store information from the control plane
type Endpoints struct {
	Svcname string   `json:"Svcname"`
	Ips     []string `json:"Ips"`
}

type endpointsMap struct {
	mu        sync.Mutex
	endpoints map[string][]string
}

func newEndpointsMap() *endpointsMap {
	return &endpointsMap{mu: sync.Mutex{}, endpoints: make(map[string][]string)}
}

func (em *endpointsMap) Get(svc string) []string {
	em.mu.Lock()
	defer em.mu.Unlock()
	return em.endpoints[svc]
}

func (em *endpointsMap) Put(svc string, backends []string) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.endpoints[svc] = backends
}

type backendSrvMap struct {
	mu sync.Mutex
	mp map[string][]BackendSrv
}

func newBackendSrvMap() *backendSrvMap {
	return &backendSrvMap{mu: sync.Mutex{}, mp: make(map[string][]BackendSrv)}
}

func (bm *backendSrvMap) Get(svc string) []BackendSrv {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	return bm.mp[svc]
}

func (bm *backendSrvMap) Put(svc string, backends []BackendSrv) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.mp[svc] = backends
}

func (bm *backendSrvMap) Incr(svc, ip string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for ind := range bm.mp[svc] {
		if bm.mp[svc][ind].Ip == ip {
			bm.mp[svc][ind].Reqs++
		}
	}
}

func (bm *backendSrvMap) Decr(svc, ip string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	for ind := range bm.mp[svc] {
		if bm.mp[svc][ind].Ip == ip {
			bm.mp[svc][ind].Reqs--
		}
	}
}

var (
	RedirectUrl_g       string
	Svc2BackendSrvMap_g = newBackendSrvMap() // holds all backends for services
	Endpoints_g         = newEndpointsMap()  // all endpoints for all services
	SvcList_g           = make([]string, 0)  // knows all service names
	// Svc2BackendSrvMap_g = make(map[string][]BackendSrv) // holds all backends for services
	// Endpoints_g         = make(map[string][]string) // all endpoints for all services
)

const (
	CLIENTPORT  = ":5000"
	PROXYINPORT = ":62081" // which port will the reverse proxy use for making outgoing request
	PROXOUTPORT = ":62082" // which port the reverse proxy listens on
)
