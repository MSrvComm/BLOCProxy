package loadbalancer

import (
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

func leasttime(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Time used")

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		log.Println("LeastTime:", err.Error())
		return nil, err
	}

	minRTT := float64(MaxInt)
	var b *globals.BackendSrv

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	it := index

	rcvTime := time.Duration(backends[it].RcvTime)
	var predTime float64
	reqs := uint64(backends[it].Reqs)
	if reqs == 0 {
		predTime = float64(System_rtt_avg_g)
	} else {
		predTime = float64(reqs+1)*backends[it].WtAvgRTT - float64(rcvTime)
	}

	it = (it + 1) % ln
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
			predTime = float64(System_rtt_avg_g)
		} else {
			predTime = float64(reqs+1)*backends[it].WtAvgRTT - float64(rcvTime)
		}
		it = (it + 1) % ln
	}
	return b, nil
}
