package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/pkg/backends"
	"github.com/MSrvComm/MiCoProxy/pkg/config"
	"github.com/MSrvComm/MiCoProxy/pkg/controllercomm"
)

const BitsPerWord = 32 << (^uint(0) >> 63)
const MaxInt = 1<<(BitsPerWord-1) - 1

type LoadBalancer struct {
	conf *config.Config
}

func NewLoadBalancer(conf *config.Config) *LoadBalancer {
	return &LoadBalancer{conf: conf}
}

func (lb *LoadBalancer) AddService(svc string) {
	if !lb.conf.SvcExists(svc) {
		ips := controllercomm.GetEndpoints(svc)
		log.Println("LB: AddService: svc:", svc, ", ips:", ips)
		lb.conf.AddNewSvc(svc, *ips)
	}
}

func (lb *LoadBalancer) GetSvcList(svc string) (*[]*backends.Backend, error) {
	lb.AddService(svc)

	svcMap, ok := lb.conf.BackendMap[svc]
	if !ok {
		return nil, errors.New("no backends found")
	}
	return svcMap, nil
}

func (lb *LoadBalancer) Random(svc string) (*backends.Backend, error) {
	log.Println("Random used") // debug
	backendsArrPtr, err := lb.GetSvcList(svc)
	if err != nil {
		log.Println("Random error", err.Error()) // debug
		return nil, err
	}

	backends := *backendsArrPtr

	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)

	ln := len(backends)
	index := rand.Intn(ln)
	return backends[index], nil
}

func (lb *LoadBalancer) NextEndpoint(svc string) (*backends.Backend, error) {
	switch lb.conf.LBPolicy {
	case "Random":
		return lb.Random(svc)
	case "LeastConn":
		return lb.LeastConn(svc)
	case "MostCredits":
		return lb.MostCredits(svc)
	default:
		return nil, errors.New("no endpoint found")
	}
}
