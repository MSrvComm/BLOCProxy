package loadbalancer

import (
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

func p2cLeastTime(svc string) (*globals.BackendSrv, error) {
	log.Println("P2C Least Time used")

	backends, err := GetSvcList(svc)
	if err != nil {
		log.Println("LeastTime:", err.Error())
		return nil, err
	}

	var backend2Return *globals.BackendSrv

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index1 := rand.Intn(ln)
	index2 := rand.Intn(ln)

	rtt1 := backends[index1].WtAvgRTT
	ts1 := float64(time.Since(backends[index1].RcvTime))
	rqs1 := backends[index1].Reqs
	predTime1 := float64(rqs1+1)*rtt1 - ts1
	if predTime1 < 0 {
		predTime1 = 0
	}

	rtt2 := backends[index2].WtAvgRTT
	ts2 := float64(time.Since(backends[index2].RcvTime))
	rqs2 := backends[index2].Reqs
	predTime2 := float64(rqs2+1)*rtt2 - ts2
	if predTime2 < 0 {
		predTime2 = 0
	}

	if predTime1 < predTime2 {
		backend2Return = &backends[index1]
	} else {
		backend2Return = &backends[index2]
	}

	return backend2Return, nil
}
