package loadbalancer

import (
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

func p2cLeastTime(svc string) (*globals.BackendSrv, error) {
	log.Println("P2C Least Time used")

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("LeastTime:", err.Error())
		return nil, err
	}

	var backend2Return *globals.BackendSrv

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index1 := rand.Intn(ln)
	for {
		// backend can be scheduled to if there are no active requests on it
		if backends[index1].NoSched && backends[index1].Reqs == 0 {
			backends[index1].NoSched = false
		} else if backends[index1].NoSched {
			index1 = rand.Intn(ln)
		} else {
			break
		}
	}

	index2 := rand.Intn(ln)
	for {
		// backend can be scheduled to if there are no active requests on it
		if backends[index2].NoSched && backends[index2].Reqs == 0 {
			backends[index2].NoSched = false
		} else if backends[index2].NoSched {
			index2 = rand.Intn(ln)
		} else {
			break
		}
	}

	rtt1 := backends[index1].WtAvgRTT
	ts1 := float64(time.Since(backends[index1].RcvTime))
	rqs1 := backends[index1].Wt
	predTime1 := float64(rqs1+1)*rtt1 - ts1
	if predTime1 < 0 {
		predTime1 = 0
	}

	rtt2 := backends[index2].WtAvgRTT
	ts2 := float64(time.Since(backends[index2].RcvTime))
	rqs2 := backends[index2].Wt
	predTime2 := float64(rqs2+1)*rtt2 - ts2
	if predTime2 < 0 {
		predTime2 = 0
	}

	if predTime1 < predTime2 {
		backend2Return = &backends[index1]
	} else {
		backend2Return = &backends[index2]
	}

	// are we waiting too long for a response?
	ts := float64(time.Since(backend2Return.RcvTime))
	rtt := backend2Return.WtAvgRTT
	rqs := backend2Return.Reqs

	// rqs == 0, ends up being a probe
	// rqs != 0 is a backend overloaded
	if rqs == 0 {
		rqs = 1 // we don't want to compare `ts` against 0 in the next step
	}
	if rtt != 0 && ts > RQS_THRESHOLD*(rtt*float64(rqs)) {
		backend2Return.NoSched = true
	}

	// is response becoming too slow?
	lastRtt := backend2Return.LastRTT

	// if rqs != 0 && lastRtt > RTT_THRESHOLD*System_rtt_avg_g {
	if rqs != 0 && float64(lastRtt) > RTT_THRESHOLD*rtt {
		backend2Return.NoSched = true
	}

	return backend2Return, nil
}
