package main

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

type BackendSrv struct {
	ip       string // ip of an endpoint
	reqs     int64  // outstanding number of request
	rcvTime  uint64 // when the last request was received
	lastRTT  uint64
	wtAvgRTT uint64
	start    uint64 // start of the hash range assigned to this node
	end      uint64 // end of the hash range assigned to this node
	grp      bool   // whether this backend is part of the main group or not
}

var (
	// change this to change load balancing policy
	// possible values are:
	// "Random"
	// "RoundRobin"
	// "LeastConn"
	// "LeastTime"
	// "RangeHash" and "RangeHashGreedy"
	g_defaultLBPolicy   = "LeastConn"
	g_Svc2BackendSrvMap = make(map[string][]BackendSrv)
	g_lastSelections    sync.Map
)

func getBackendSvcList(svc string) ([]BackendSrv, error) {
	mapExists := g_Svc2BackendSrvMap[svc][:] // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		return mapExists, nil
	}
	// else if
	// make entries into backendSrvs here
	var backendSrvs []BackendSrv
	ips := g_endpoints[svc]
	if len(ips) > 0 {
		for _, ip := range ips {
			// backendSrvs = append(backendSrvs, BackendSrv{ip: ip, reqs: 0, lastRTT: 0, avgRTT: 0})
			backendSrvs = append(backendSrvs, BackendSrv{ip: ip, reqs: 0, lastRTT: 0, grp: true})
		}
		// call the hash distribution service here
		hashDistribution(&backendSrvs, len(ips))
		// add backend to the backend maps
		g_Svc2BackendSrvMap[svc] = backendSrvs
		return g_Svc2BackendSrvMap[svc][:], nil
	}
	// else
	return nil, errors.New("no backend found")
}

// initialize the seed only once
const BitsPerWord = 32 << (^uint(0) >> 63)
const MaxInt = 1<<(BitsPerWord-1) - 1

var seed = time.Now().UTC().UnixNano()

func RoundRobin(svc string) (*BackendSrv, error) {
	log.Println("Round Robin used") // debug
	// we store index as 1 to N
	// 0 indicates the absence of svc
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	l := len(backends)

	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ind, ok := g_lastSelections.Load(svc)
	var index int
	if !ok {
		index = rand.Intn(l)
	} else {
		index = ind.(int)
	}

	backend := &backends[index]
	index++
	index = index % l
	g_lastSelections.Store(svc, index)
	return backend, nil
}

func LeastConn(svc string) (*BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	// P2C Least Conn
	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)
	srv1 := &backends[rand.Intn(len(backends))]
	srv2 := &backends[rand.Intn(len(backends))]

	// var ip string
	if srv1.reqs < srv2.reqs {
		return srv1, nil
	}
	return srv2, nil
}

func LeastTime(svc string) (*BackendSrv, error) {
	log.Println("Least Time used") // debug
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	minRTT := uint64(MaxInt)
	var b *BackendSrv

	seed = time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	it := index
	// the do part of the do-while logic
	rcvTime := time.Duration(backends[it].rcvTime)
	var predTime uint64
	reqs := uint64(backends[it].reqs)
	if reqs == 0 {
		predTime = uint64(rcvTime)
	} else {
		predTime = (reqs+1)*backends[it].wtAvgRTT - uint64(rcvTime)
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
		rcvTime = time.Duration(backends[it].rcvTime)

		if reqs == 0 {
			predTime = uint64(rcvTime)
		} else {
			predTime = (reqs+1)*backends[it].wtAvgRTT - uint64(rcvTime)
		}
		it = (it + 1) % ln
	}

	return b, nil
}

func Random(svc string) (*BackendSrv, error) {
	log.Println("Random used") // debug
	backends, err := getBackendSvcList(svc)
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

func NextEndpoint(svc string) (*BackendSrv, error) {
	switch g_defaultLBPolicy {
	case "RoundRobin":
		return RoundRobin(svc)
	case "LeastConn":
		return LeastConn(svc)
	case "LeastTime":
		return LeastTime(svc)
	case "Random":
		return Random(svc)
	case "RangeHash":
		return rangeHashRounds(svc)
	case "RangeHashGreedy":
		return rangeHashGreedy(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
	// return RoundRobin(svc)
}
