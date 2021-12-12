package main

import (
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"
)

type BackendSrv struct {
	ip   string // ip of an endpoint
	reqs int64  // outstanding number of request
	// rcvTime  time.Time // when the last request was received
	rcvTime  int64 // when the last request was received
	lastRTT  int64
	avgRTT   int64
	wtAvgRTT int64
}

var (
	// change this to change load balancing policy
	// possible values are "RoundRobin", "LeastConn", "LeastTime" and to be defined ... TODO:
	defaultLBPolicy   = "RoundRobin"
	Svc2BackendSrvMap = make(map[string][]BackendSrv)
	// lastSelections    = make(map[string]int)
	lastSelections sync.Map
)

func getBackendSvcList(svc string) ([]BackendSrv, error) {
	mapExists := Svc2BackendSrvMap[svc][:] // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		return mapExists, nil
	}
	// else if
	// make entries into backendSrvs here
	var backendSrvs []BackendSrv
	ips := endpoints[svc]
	if len(ips) > 0 {
		for _, ip := range ips {
			backendSrvs = append(backendSrvs, BackendSrv{ip: ip, reqs: 0, lastRTT: 0, avgRTT: 0})
		}
		// add backend to the backend maps
		Svc2BackendSrvMap[svc] = backendSrvs
		return Svc2BackendSrvMap[svc][:], nil
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
	// should have also covered l == 0
	if err != nil {
		return nil, err
	}

	l := len(backends)

	ind, ok := lastSelections.Load(svc)
	var index int
	// index, ok := lastSelections[svc]
	if !ok {
		// index = 0
		index = rand.Intn(l)
	} else {
		index = ind.(int)
	}

	backend := &backends[index]
	index++
	index = index % l
	lastSelections.Store(svc, index)
	// lastSelections[svc] = index
	return backend, nil
}

func LeastConn(svc string) (*BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	// minReq := int64(math.MaxInt64)
	// var b *BackendSrv

	// for i := range backends {
	// 	if backends[i].reqs < minReq {
	// 		minReq = backends[i].reqs
	// 		b = &backends[i]
	// 	}
	// }
	// return b, nil

	// P2C Least Conn
	if seed == MaxInt {
		seed = time.Now().UTC().UnixNano()
	}
	seed += 1
	rand.Seed(seed)
	srv1 := &backends[rand.Intn(len(backends))]
	srv2 := &backends[rand.Intn(len(backends))]

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
	minRTT := time.Hour.Nanoseconds()
	var b *BackendSrv

	for i := range backends {
		rcvTime := time.Duration(backends[i].rcvTime)
		// predTime := backends[i].reqs*backends[i].wtAvgRTT - int64(time.Since(backends[i].rcvTime))
		predTime := backends[i].reqs*backends[i].wtAvgRTT - int64(rcvTime)
		if predTime < minRTT {
			minRTT = predTime
			b = &backends[i]
		}
	}
	return b, nil
}

func NextEndpoint(svc string) (*BackendSrv, error) {
	switch defaultLBPolicy {
	case "RoundRobin":
		return RoundRobin(svc)
	case "LeastConn":
		return LeastConn(svc)
	case "LeastTime":
		return LeastTime(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
	// return RoundRobin(svc)
}
