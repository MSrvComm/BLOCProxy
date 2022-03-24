package globals

import "sync"

type BackendSrv struct {
	Ip       string // ip of an endpoint
	Reqs     int64  // outstanding number of request
	RcvTime  uint64 // when the last request was received
	LastRTT  uint64
	WtAvgRTT uint64
	Start    uint64 // start of the hash range assigned to this node
	End      uint64 // end of the hash range assigned to this node
	Grp      bool   // whether this backend is part of the main group or not
}

type Endpoint struct {
	Svcname string   `json:"Svcname"`
	Ips     []string `json:"Ips"`
}

var (
	// change this to change load balancing policy
	// possible values are:
	// "Random"
	// "RoundRobin"
	// "LeastConn"
	// "LeastTime"
	// "RangeHash" and "RangeHashGreedy"
	// "Global"
	DefaultLBPolicy_g   = "LeastConn"
	Svc2BackendSrvMap_g = make(map[string][]BackendSrv)
	LastSelections_g    sync.Map
	SvcList_g           = []string{""} // names of all services
	// Endpoints_g         sync.Map
	RedirectUrl_g       string
	GlobalMap_g         sync.Map
	Endpoints_g         = make(map[string][]string) // all endpoints for all services
)

const (
	CLIENTPORT   = ":5000"
	PROXYINPORT  = ":62081" // which port will the reverse proxy use for making outgoing request
	PROXYOUTPORT = ":62082" // which port the reverse proxy listens on
)

// used for timing
type PathStats struct {
	Count    uint64
	RTT      uint64
	WtAvgRTT uint64
}
