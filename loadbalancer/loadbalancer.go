package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

const BitsPerWord = 32 << (^uint(0) >> 63)
const MaxInt = 1<<(BitsPerWord-1) - 1

func GetBackendSvcList(svc string) ([]globals.BackendSrv, error) {
	mapExists := globals.Svc2BackendSrvMap_g.Get(svc) // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		return mapExists, nil
	}
	// else if
	// make entries into backendSrvs here
	var backendSrvs []globals.BackendSrv
	ips := globals.Endpoints_g.Get(svc)
	if len(ips) > 0 {
		for _, ip := range ips {
			backendSrvs = append(backendSrvs, globals.BackendSrv{Ip: ip, Reqs: 0, LastRTT: 0})
		}
		// call the hash distribution service here
		hashDistribution(&backendSrvs, len(ips))
		// add backend to the backend maps
		globals.Svc2BackendSrvMap_g.Put(svc, backendSrvs)
		return globals.Svc2BackendSrvMap_g.Get(svc), nil
	}
	// else
	return nil, errors.New("no backends found")
}

func LeastConn(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	// P2C Least Conn
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	srv1 := &backends[rand.Intn(len(backends))]
	srv2 := &backends[rand.Intn(len(backends))]

	// var ip string
	if srv1.Reqs < srv2.Reqs {
		// log.Println("LeastConn: backend selected: ", srv1)
		return srv1, nil
	}
	// log.Println("LeastConn: backend selected: ", srv2)
	return srv2, nil
}

func NextEndpoint(svc string) (*globals.BackendSrv, error) {
	switch globals.DefaultLBPolicy_g {
	case "LeastConn":
		return LeastConn(svc)
	case "RangeHash":
		return rangeHashGreedy(svc)
	case "RangeHashRounds":
		return rangeHashRounds(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
}
