package loadbalancer

import (
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

const (
	RQS_THRESHOLD = 1.5 // how stuck will be let the servers be
	RTT_THRESHOLD = 5   // how bad will we let the response times to become
)

func leasttime(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Time used")

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("LeastTime:", err.Error())
		return nil, err
	}

	var backend2Return *globals.BackendSrv

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	it := index

	// we try to predict when we will receive the response
	// select the backend we think will provide the earliest response
	//
	// we score each backend on two counts
	// number of active request we know of (rqs) * avg response time from that server (rtt)
	// time since we last sent the server a request (ts)
	// score := rqs * rtt + (1 / ts)
	// this is the normal operating mode of the algorithm
	//
	// however, when ts >= 1.25(rtt*rqs) : rqs > 0, we assume there is something wrong with the backend and stop scheduling to it
	// if rqs == 0 && ts >= 1.5(rtt), we send the next request to the backend as a probe and then take the backend out of scheduling
	var predTime float64
	minRTT := float64(MaxInt)

	for {
		rtt := backends[it].WtAvgRTT
		ts := float64(time.Since(backends[it].RcvTime))
		// rqs := backends[it].Reqs

		// modulate number of requests for the backend by weight
		// if we have been sending more requests than others, this is adjusted downwards and vice versa
		rqs := float64(backends[it].Reqs+1) * backends[it].WtAvgRTT

		// predTime = float64(rqs+1)*rtt - ts
		predTime = (rqs+1)*rtt - ts

		if predTime < 0 {
			predTime = 0
		}

		if predTime < minRTT {
			minRTT = predTime
			backend2Return = &backends[it]
		}

		it = (it + 1) % ln
		if it == index {
			break
		}
	}

	return backend2Return, nil
}
