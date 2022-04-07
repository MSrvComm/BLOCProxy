package loadbalancer

import (
	"fmt"
	"hash/crc64"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

const (
	RESPONSE_TIME_THRESHOLD = 0.1
	PRIME                   = 7
	SKEW_THRESHOLD          = 0.2
	SIZE_THRESHOLD          = 0.3
)

var (
	System_rtt_avg_g = uint64(0)
	System_reqs_g    = uint64(0)
	system_range_g   = uint64(math.Pow(2, PRIME) - 1)
)

func hash(s string) uint64 {
	return crc64.Checksum([]byte(s), crc64.MakeTable(crc64.ISO)) % system_range_g
}

func hashDistribution(backendSrvs *[]globals.BackendSrv, n int) {
	nodeDefault := system_range_g / uint64(n)
	start := uint64(0)
	end := uint64(nodeDefault)
	for i := 0; i < n; i++ {
		(*backendSrvs)[i].Start = start
		(*backendSrvs)[i].End = end
		start = end + 1
		end = start + nodeDefault
	}
}

func rangeHashGreedy(svc string) (*globals.BackendSrv, error) {
	log.Println("Range Hash Greedy used") // debug
	// generate a random hash for every request
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	reqHash := hash(ip)
	// reqHash := rand.Int63() % int64(system_range_g)

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	backend2return := &backends[0]
	for i := range backends {
		if uint64(reqHash) >= (&backends[i]).Start && uint64(reqHash) <= (&backends[i]).End {
			backend2return = &backends[i]
		}
	}

	// greedy - redistribute on every request
	redistributeHash(svc)

	return backend2return, nil
}

func rangeHashRounds(svc string) (*globals.BackendSrv, error) {
	log.Println("Range Hash Rounds")
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	ip := fmt.Sprintf("%d.%d.%d.%d", rand.Intn(255), rand.Intn(255), rand.Intn(255), rand.Intn(255))
	reqHash := hash(ip)
	// reqHash := rand.Int63n(int64(system_range_g))

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("Range Hash Rounds:", err.Error())
		return nil, err
	}

	backendsNotInMainGrp := 0
	backend2Return := &backends[0]

	// if we redistribute the hash in this call
	// we only want to do it once
	redistributed := false

	for i := range backends {
		// check for the skew threshold
		// whether the response time of the pod is too high or too low
		wtAvgFlt := float64((&backends[i]).WtAvgRTT)
		sysAvgFlt := float64(System_rtt_avg_g)
		if wtAvgFlt >= (1+SKEW_THRESHOLD)*sysAvgFlt || wtAvgFlt <= (1-SKEW_THRESHOLD)*sysAvgFlt {
			redistributeHash(svc)
			redistributed = true
		}
		// backend in main group but fails the Response time threshold condition
		if !redistributed && (&backends[i]).Grp && wtAvgFlt >= (1+RESPONSE_TIME_THRESHOLD)*sysAvgFlt {
			(&backends[i]).Grp = false
			backendsNotInMainGrp++
		}
		// backend not in main group
		if !redistributed && !(&backends[i]).Grp {
			// the response time threshold is still failed
			if float64((&backends[i]).WtAvgRTT) >= (1+RESPONSE_TIME_THRESHOLD)*float64(System_rtt_avg_g) {
				backendsNotInMainGrp++
			} else if float64((&backends[i]).WtAvgRTT) < (1+RESPONSE_TIME_THRESHOLD)*float64(System_rtt_avg_g) {
				backendsNotInMainGrp++
			} else {
				// add it back to the main group
				(&backends[i]).Grp = true
			}
		}
		// log.Printf("backend %s -> start: %d, end: %d", (&backends[i]).ip, (&backends[i]).start, (&backends[i]).end) // debug
		if uint64(reqHash) >= (&backends[i]).Start && uint64(reqHash) <= (&backends[i]).End {
			backend2Return = &backends[i]
		}
	}
	// check if reassignment of hash range is required
	if !redistributed && float64(backendsNotInMainGrp) >= float64(len(backends))*(1+SIZE_THRESHOLD) {
		redistributeHash(svc)
	}
	return backend2Return, nil
}

func redistributeHash(svc string) {
	total := float64(0)
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("Redistribute Hash:", err.Error())
		return
	}

	// calculate the normalisation
	for i := range backends {
		rtt := (&backends[i]).WtAvgRTT + 1 // can overflow, otherwise protects against division by 0
		total += 1 / (float64(rtt) + 1)    // shift rtt inverse values towards 1 so that 'ratio', later, is not 0
	}
	// redistribute the hashranges
	nodeRangeStart := uint64(0)
	for i := range backends {
		rttInv := 1 / (float64((&backends[i]).WtAvgRTT) + 1) // protect against division by 0
		ratio := rttInv / total
		rs := ratio * float64(system_range_g)
		nodeRange := uint64(rs)
		// log.Printf("wtAvgRTT: %v, NodeRange: %v, total: %v, ratio: %v, rs: %v", (&backends[i]).wtAvgRTT, nodeRange, total, ratio, rs) // debug
		atomic.StoreUint64(&backends[i].Start, nodeRangeStart)
		end := nodeRangeStart + nodeRange
		atomic.StoreUint64(&backends[i].End, end)
		nodeRangeStart = end + 1
	}
}
