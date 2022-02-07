package main

import (
	"hash/crc64"
	"log"
	"math"
)

const (
	RESPONSE_TIME_THRESHOLD = 0.1
	SIZE_THRESHOLD          = 0.3
	SKEW_THRESHOLD          = 0.2
	PRIME                   = 111
)

var (
	system_range          = uint64(math.Pow(2, PRIME) - 1)
	system_rtt_avg uint64 = 0
	sys_reqs       int64  = 0
)

func hash(s string) uint64 {
	return crc64.Checksum([]byte(s), crc64.MakeTable(crc64.ISO)) % system_range
}

func hashDistribution(backendSrvs *[]BackendSrv, n int) {
	nodeDefault := system_range / uint64(n)
	start := uint64(0)
	for i := 0; i < n; i++ {
		end := uint64(start) + nodeDefault
		(*backendSrvs)[i].start = start
		(*backendSrvs)[i].end = start + nodeDefault
		start = end + 1
	}
}

func rangeHashLB(svc, headers string) (*BackendSrv, error) {
	log.Println("Range Hash used") // debug
	hsh := hash(headers)
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	backendsNotInGrp := 0
	backend2return := &backends[0]
	// if we redistribute the range in this call
	// we want to do it only once
	redistributed := false

	for _, backend := range backends {
		// check for the skew threshold
		if float64(backend.wtAvgRTT) >= (1+SKEW_THRESHOLD)*float64(system_rtt_avg) {
			redistributeHash(svc)
			redistributed = true
		}
		// backend in main group but fails the Response time threshold condition
		if !redistributed && backend.grp && float64(backend.wtAvgRTT) >= (1+RESPONSE_TIME_THRESHOLD)*float64(system_rtt_avg) {
			backend.grp = false
			backendsNotInGrp++
		}
		// backend not in main group
		if !redistributed && !backend.grp {
			// the response time threshold is still failed
			if float64(backend.wtAvgRTT) >= (1+RESPONSE_TIME_THRESHOLD)*float64(system_rtt_avg) {
				backendsNotInGrp++
			} else {
				// add it back to the main group
				backend.grp = true
			}
		}
		if hsh >= backend.start && hsh <= backend.end {
			backend2return = &backend
		}
	}

	// check if reassignment of hash range is required
	if !redistributed && float64(backendsNotInGrp) >= float64(len(backends))*(1+SIZE_THRESHOLD) {
		redistributeHash(svc)
	}

	return backend2return, nil
}

func redistributeHash(svc string) {
	total := float64(0)
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return
	}

	// calculate the normalisation
	for _, backend := range backends {
		total += 1 / float64(backend.wtAvgRTT)
	}
	// redistribute the hashranges
	nodeRangeStart := uint64(0)
	for _, backend := range backends {
		nodeRange := uint64((1 / float64(backend.wtAvgRTT)) / total)
		backend.start = nodeRangeStart
		end := nodeRangeStart + nodeRange
		backend.end = end
		nodeRangeStart = end + 1
	}
}
