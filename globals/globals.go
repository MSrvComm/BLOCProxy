package globals

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/incoming"
)

// BackendSrv stores information for internal decision making
type BackendSrv struct {
	RW              *sync.RWMutex
	Ip              string
	Reqs            int64
	RcvTime         time.Time
	LastRTT         uint64
	WtAvgRTT        float64
	CreditsBackend  int64 // credits we received
	CreditsFrontend int64 // credits allocated by us
}

func (backend *BackendSrv) ChangeCredit(n int64) {
	atomic.AddInt64(&backend.CreditsFrontend, n)
}

func (backend *BackendSrv) Backoff() {
	backend.RW.Lock()
	defer backend.RW.Unlock()
	backend.RcvTime = time.Now() // now time since > globals.RESET_INTERVAL; refer to MLeastConn algo
	backend.CreditsBackend = 0
}

func (backend *BackendSrv) Incr() {
	backend.RW.Lock()
	defer backend.RW.Unlock()
	backend.Reqs++
}

func (backend *BackendSrv) Decr() {
	backend.RW.Lock()
	defer backend.RW.Unlock()
	// we use up a credit whenever a new request is sent to that backend
	backend.CreditsBackend--
	backend.Reqs--
}

func (backend *BackendSrv) Update(start time.Time, credits int64, elapsed uint64) {
	backend.RW.Lock()
	defer backend.RW.Unlock()
	backend.RcvTime = start
	backend.LastRTT = elapsed
	backend.WtAvgRTT = backend.WtAvgRTT*0.5 + 0.5*float64(elapsed)
	backend.CreditsBackend = credits
}

// Endpoints store information from the control plane
type Endpoints struct {
	Svcname string   `json:"Svcname"`
	Ips     []string `json:"Ips"`
}

type endpointsMap struct {
	mu        *sync.Mutex
	endpoints map[string][]string
}

func newEndpointsMap() *endpointsMap {
	return &endpointsMap{mu: &sync.Mutex{}, endpoints: make(map[string][]string)}
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
	mu *sync.RWMutex
	mp map[string][]BackendSrv
}

func newBackendSrvMap() *backendSrvMap {
	return &backendSrvMap{mu: &sync.RWMutex{}, mp: make(map[string][]BackendSrv)}
}

func (bm *backendSrvMap) Get(svc string) []BackendSrv {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.mp[svc]
}

func (bm *backendSrvMap) Put(svc string, backends []BackendSrv) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.mp[svc] = backends
}

func (bm *backendSrvMap) SearchByHostIP(ip string) *BackendSrv {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	for _, arr := range bm.mp {
		for i := range arr {
			if arr[i].Ip == ip {
				return &arr[i]
			}
		}
	}
	return nil
}

var (
	RedirectUrl_g       string
	Svc2BackendSrvMap_g = newBackendSrvMap() // holds all backends for services
	Endpoints_g         = newEndpointsMap()  // all endpoints for all services
	SvcList_g           = make([]string, 0)  // knows all service names
	Downstream_svc_g    string
	InProxy             *incoming.Proxy
)

const (
	CLIENTPORT     = ":5000"
	CREDIPORT      = ":5001"     // port on which the credit system is listening
	PROXYINPORT    = ":62081"    // which port will the reverse proxy use for making outgoing request
	PROXOUTPORT    = ":62082"    // which port the reverse proxy listens on
	RESET_INTERVAL = time.Second // interval after which credit info of backend expires
)
