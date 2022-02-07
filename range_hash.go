package main

import (
	"fmt"
	"hash/crc64"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
)

const (
	RESPONSE_TIME_THRESHOLD = 0.1
	SIZE_THRESHOLD          = 0.3
	SKEW_THRESHOLD          = 0.2
	PRIME                   = 7
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

func rangeHashGreedy(svc string) (*BackendSrv, error) {
	log.Println("Range Hash Greedy used") // debug
	// generate a random hash for every request
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	hsh := hash(ip)

	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	backend2return := &backends[0]
	for i := range backends {
		if hsh >= (&backends[i]).start && hsh <= (&backends[i]).end {
			backend2return = &backends[i]
		}
	}

	// greedy - redistribute on every request
	redistributeHash(svc)

	return backend2return, nil
}

func rangeHashRounds(svc string) (*BackendSrv, error) {
	log.Println("Range Hash used") // debug
	// generate a random hash for every request
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	hsh := hash(ip)

	backends, err := getBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	backendsNotInGrp := 0
	backend2return := &backends[0]
	// if we redistribute the range in this call
	// we want to do it only once
	redistributed := false

	for i := range backends {
		// check for the skew threshold
		// whether the response time of the pod is too high or too low
		if float64((&backends[i]).wtAvgRTT) >= (1+SKEW_THRESHOLD)*float64(system_rtt_avg) || float64((&backends[i]).wtAvgRTT) <= (1-SKEW_THRESHOLD)*float64(system_rtt_avg) {
			redistributeHash(svc)
			redistributed = true
		}
		// backend in main group but fails the Response time threshold condition
		if !redistributed && (&backends[i]).grp && float64((&backends[i]).wtAvgRTT) >= (1+RESPONSE_TIME_THRESHOLD)*float64(system_rtt_avg) {
			(&backends[i]).grp = false
			backendsNotInGrp++
		}
		// backend not in main group
		if !redistributed && !(&backends[i]).grp {
			// the response time threshold is still failed
			if float64((&backends[i]).wtAvgRTT) >= (1+RESPONSE_TIME_THRESHOLD)*float64(system_rtt_avg) {
				backendsNotInGrp++
			} else if float64((&backends[i]).wtAvgRTT) < (1+RESPONSE_TIME_THRESHOLD)*float64(system_rtt_avg) {
				backendsNotInGrp++
			} else {
				// add it back to the main group
				(&backends[i]).grp = true
			}
		}
		// log.Printf("backend %s -> start: %d, end: %d", (&backends[i]).ip, (&backends[i]).start, (&backends[i]).end) // debug
		if hsh >= (&backends[i]).start && hsh <= (&backends[i]).end {
			backend2return = &backends[i]
		}
	}
	// check if reassignment of hash range is required
	if !redistributed && float64(backendsNotInGrp) >= float64(len(backends))*(1+SIZE_THRESHOLD) {
		redistributeHash(svc)
	}

	// // greedy - redistribute on every request
	// redistributeHash(svc)

	return backend2return, nil
}

func redistributeHash(svc string) {
	// log.Println("redistributeHash called") // debug
	total := float64(0)
	backends, err := getBackendSvcList(svc)
	if err != nil {
		return
	}

	// calculate the normalisation
	for i := range backends {
		rtt := (&backends[i]).wtAvgRTT + 1 // can overflow, otherwise protects against division by 0
		total += 1 / (float64(rtt) + 1) // shift rtt inverse values towards 1 so that 'ratio', later, is not 0
	}
	// redistribute the hashranges
	nodeRangeStart := uint64(0)
	for i := range backends {
		rttInv := 1 / (float64((&backends[i]).wtAvgRTT) + 1) // protect against division by 0
		ratio := rttInv / total
		rs := ratio * float64(system_range)
		nodeRange := uint64(rs)
		// log.Printf("wtAvgRTT: %v, NodeRange: %v, total: %v, ratio: %v, rs: %v", (&backends[i]).wtAvgRTT, nodeRange, total, ratio, rs) // debug
		atomic.StoreUint64(&backends[i].start, nodeRangeStart)
		end := nodeRangeStart + nodeRange
		atomic.StoreUint64(&backends[i].end, end)
		nodeRangeStart = end + 1
	}
}
