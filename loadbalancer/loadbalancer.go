package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/redisops"
)

// type BackendSrv struct {
// 	Ip       string // ip of an endpoint
// 	Reqs     int64  // outstanding number of request
// 	RcvTime  uint64 // when the last request was received
// 	LastRTT  uint64
// 	WtAvgRTT uint64
// 	Start    uint64 // start of the hash range assigned to this node
// 	End      uint64 // end of the hash range assigned to this node
// 	Grp      bool   // whether this backend is part of the main group or not
// }

// var (
// 	// change this to change load balancing policy
// 	// possible values are:
// 	// "Random"
// 	// "RoundRobin"
// 	// "LeastConn"
// 	// "LeastTime"
// 	// "RangeHash" and "RangeHashGreedy"
// 	// "Global"
// 	defaultLBPolicy_g   = "LeastConn"
// 	Svc2BackendSrvMap_g = make(map[string][]BackendSrv)
// 	lastSelections_g    sync.Map
// )

func GetBackendSvcList(svc string) ([]globals.BackendSrv, error) {
	mapExists := globals.Svc2BackendSrvMap_g[svc][:] // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		return mapExists, nil
	}
	// else if
	// make entries into backendSrvs here
	var backendSrvs []globals.BackendSrv
	ips := globals.Endpoints_g[svc]
	// ips_a, _ := globals.Endpoints_g.Load(svc)
	// ips := ips_a.([]string)
	if len(ips) > 0 {
		for _, ip := range ips {
			// backendSrvs = append(backendSrvs, BackendSrv{ip: ip, reqs: 0, lastRTT: 0, avgRTT: 0})
			backendSrvs = append(backendSrvs, globals.BackendSrv{Ip: ip, Reqs: 0, LastRTT: 0, Grp: true})
		}
		// call the hash distribution service here
		// hashDistribution(&backendSrvs, len(ips))
		// add backend to the backend maps
		globals.Svc2BackendSrvMap_g[svc] = backendSrvs
		return globals.Svc2BackendSrvMap_g[svc][:], nil
	}
	// else
	return nil, errors.New("no backend found")
}

// initialize the seed only once
const BitsPerWord = 32 << (^uint(0) >> 63)
const MaxInt = 1<<(BitsPerWord-1) - 1

var seed = time.Now().UTC().UnixNano()

func RoundRobin(svc string) (*globals.BackendSrv, error) {
	log.Println("Round Robin used") // debug
	// we store index as 1 to N
	// 0 indicates the absence of svc
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	l := len(backends)

	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ind, ok := globals.LastSelections_g.Load(svc)
	var index int
	if !ok {
		index = rand.Intn(l)
	} else {
		index = ind.(int)
	}

	backend := &backends[index]
	index++
	index = index % l
	globals.LastSelections_g.Store(svc, index)
	return backend, nil
}

func LeastConn(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	// P2C Least Conn
	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)
	srv1 := &backends[rand.Intn(len(backends))]
	srv2 := &backends[rand.Intn(len(backends))]

	// var ip string
	if srv1.Reqs < srv2.Reqs {
		return srv1, nil
	}
	return srv2, nil
}

func LeastTime(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Time used") // debug
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	minRTT := uint64(MaxInt)
	var b *globals.BackendSrv

	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	it := index
	// the do part of the do-while logic
	rcvTime := time.Duration(backends[it].RcvTime)
	var predTime uint64
	reqs := uint64(backends[it].Reqs)
	if reqs == 0 {
		predTime = uint64(rcvTime)
	} else {
		predTime = (reqs+1)*backends[it].WtAvgRTT - uint64(rcvTime)
	}
	it = (it + 1) % ln
	// the while part
	for {
		if predTime < minRTT {
			minRTT = predTime
			b = &backends[it]
		}
		if it == index {
			break
		}
		rcvTime = time.Duration(backends[it].RcvTime)

		if reqs == 0 {
			predTime = uint64(rcvTime)
		} else {
			predTime = (reqs+1)*backends[it].WtAvgRTT - uint64(rcvTime)
		}
		it = (it + 1) % ln
	}

	return b, nil
}

func Random(svc string) (*globals.BackendSrv, error) {
	log.Println("Random used") // debug
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("Random error", err.Error()) // debug
		return nil, err
	}

	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	return &backends[index], nil
}

// uses the redis service to get global data
func Global(svc string) (string, error) {
	backends, err := redisops.Retrieve(svc)
	if err != nil {
		return "", err
	}
	var minReqs int64 = MaxInt
	var backend2Return string
	for backend, reqs := range backends {
		if reqs < minReqs {
			backend2Return = backend
			minReqs = reqs
		}
	}
	log.Println("Global", backend2Return) // debug
	return backend2Return, nil
}

func P2CGlobal(svc string) (string, error) {
	backends, err := redisops.Retrieve(svc)
	if err != nil {
		return "", err
	}

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	index1 := rand.Intn(len(backends))
	index2 := rand.Intn(len(backends))

	i := 0
	var backend1, backend2 string
	var reqs1, reqs2 int64
	for backend, reqs := range backends {
		if i == index1 {
			backend1 = backend
			reqs1 = reqs
		}
		if i == index2 {
			backend2 = backend
			reqs2 = reqs
		}
		i++
	}
	if reqs1 < reqs2 {
		return backend1, nil
	}
	return backend2, nil
}

func NextEndpoint(svc string) (*globals.BackendSrv, error) {
	switch globals.DefaultLBPolicy_g {
	case "RoundRobin":
		return RoundRobin(svc)
	case "LeastConn":
		return LeastConn(svc)
	case "LeastTime":
		return LeastTime(svc)
	case "Random":
		return Random(svc)
	// case "RangeHash":
	// 	return rangeHashRounds(svc)
	// case "RangeHashGreedy":
	// 	return rangeHashGreedy(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
	// return RoundRobin(svc)
}
