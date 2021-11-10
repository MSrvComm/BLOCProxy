package main

import (
	"errors"
	"log"
	"math"
	"time"
)

type BackendSrv struct {
	ip       string // ip of an endpoint
	reqs     int64  // outstanding number of request
	rcvTime  int64  // when the last request was received
	lastRTT  int64
	avgRTT   int64
	wtAvgRTT int64
}

var (
	// change this to change load balancing policy
	// possible values are "RoundRobin", "LeastConn", "LeastTime" and to be defined ... TODO:
	LBPolicy          = "LeastTime"
	Svc2BackendSrvMap = make(map[string][]BackendSrv)
	lastSelections    = make(map[string]int)
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

	index, ok := lastSelections[svc]
	if !ok {
		index = 0
	}

	backend := &backends[index]
	index++
	index = index % l
	lastSelections[svc] = index
	return backend, nil
}

func LeastConn(svc string) (*BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	minReq := int64(math.MaxInt64)
	var b *BackendSrv

	for i := range backends {
		if backends[i].reqs < minReq {
			minReq = backends[i].reqs
			b = &backends[i]
		}
	}
	return b, nil
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
		predTime := backends[i].reqs*backends[i].wtAvgRTT - backends[i].rcvTime
		if predTime < minRTT {
			minRTT = predTime
			b = &backends[i]
		}
	}
	return b, nil
}

func NextEndpoint(svc string) (*BackendSrv, error) {
	switch LBPolicy {
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
