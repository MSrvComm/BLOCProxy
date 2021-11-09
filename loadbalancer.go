package main

import (
	"errors"
	"log"
)

type BackendSrv struct {
	ip      string // ip of an endpoint
	reqs    int64  // outstanding number of request
	lastRTT int64
	avgRTT  int64
}

var (
	// change this to change load balancing policy
	// possible values are "RoundRobin" and to be defined ... TODO:
	LBPolicy          = "LeastConn"
	Svc2BackendSrvMap = make(map[string][]BackendSrv)
	lastSelections    = make(map[string]int)
)

func getBackendSvcList(svc string) ([]BackendSrv, error) {
	mapExists := Svc2BackendSrvMap[svc][:] // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		log.Println("map exists") // debug
		return mapExists, nil
	}
	// else if
	// make entries into backendSrvs here
	var backendSrvs []BackendSrv
	ips := endpoints[svc]
	if len(ips) > 0 {
		log.Println("ips loop") // debug
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

	log.Printf("%#+v\n", backends) // debug

	index, ok := lastSelections[svc]
	if !ok {
		index = 0
	}

	backend := &backends[index]
	index++
	index = index % l
	log.Println("saving index:", index) // debug
	lastSelections[svc] = index
	return backend, nil
}

func NextEndpoint(svc string) (*BackendSrv, error) {
	// switch LBPolicy {
	// case "RoundRobin":
	// 	return RoundRobin(svc)
	// default:
	// 	return
	// }
	return RoundRobin(svc)
}
