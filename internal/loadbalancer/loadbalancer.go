package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

var defaultLBPolicy_g string

const BitsPerWord = 32 << (^uint(0) >> 63)
const MaxInt = 1<<(BitsPerWord-1) - 1

func PopulateSvcList(svc string) bool {
	mapExists := globals.Svc2BackendSrvMap_g.Get(svc) // send a reference to the original instead of making a copy
	if len(mapExists) > 0 {
		return true
	}
	var backendSrvs []globals.BackendSrv
	ips := globals.Endpoints_g.Get(svc)
	if len(ips) > 0 {
		for _, ip := range ips {
			backendSrvs = append(backendSrvs,
				globals.BackendSrv{Ip: ip,
					Reqs:     0,
					LastRTT:  0,
					WtAvgRTT: 0,
					// credit for all backends is set to 1 at the start
					// it's up to the backend to update it
					CreditsBackend: 0,
					RcvTime: time.Now(),
				})
		}
		// add backend to the backend maps
		globals.Svc2BackendSrvMap_g.Put(svc, backendSrvs)
		time.Sleep(time.Nanosecond * 100)
		return true
	}
	return false
}

func GetSvcList(svc string) ([]globals.BackendSrv, error) {
	found := PopulateSvcList(svc)
	if found {
		return globals.Svc2BackendSrvMap_g.Get(svc), nil
	}
	return nil, errors.New("no backends found")
}

func Random(svc string) (*globals.BackendSrv, error) {
	log.Println("Random used") // debug
	backends, err := GetSvcList(svc)
	if err != nil {
		log.Println("Random error", err.Error()) // debug
		return nil, err
	}

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	return &backends[index], nil
}

func NextEndpoint(svc string) (*globals.BackendSrv, error) {
	if defaultLBPolicy_g == "" {
		defaultLBPolicy_g = os.Getenv("LBPolicy")
	}
	switch defaultLBPolicy_g {
	case "Random":
		return Random(svc)
	case "LeastConn":
		return LeastConn(svc)
	case "MLeastConn":
		return MLeastConn(svc)
	case "MLeastConnFull":
		return MLeastConnFull(svc)
	case "LeastTime":
		return leasttime(svc)
	case "P2CLeastTime":
		return p2cLeastTime(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
}
