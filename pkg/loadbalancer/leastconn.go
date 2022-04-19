package loadbalancer

import (
	"errors"
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/pkg/backends"
)

func (lb *LoadBalancer) LeastConn(svc string) (*backends.Backend, error) {
	log.Println("Least Connection used") // debug
	backends, err := lb.GetSvcList(svc)
	if err != nil {
		return nil, err
	}

	// backends := *backendsArrPtr

	if len(backends) <= 0 {
		return nil, errors.New("LeastConn: no backend found")
	}

	// P2C Least Conn
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	srv1 := backends[rand.Intn(len(backends))]
	srv2 := backends[rand.Intn(len(backends))]

	// var ip string
	if srv1.Reqs < srv2.Reqs {
		return srv1, nil
	}
	return srv2, nil
}

// var backend2Return *backends.Backend

func (lb *LoadBalancer) MostCredits(svc string) (*backends.Backend, error) {
	log.Println("Modified Least Connection used") // debug
	backends, err := lb.GetSvcList(svc)
	if err != nil {
		return nil, err
	}
	// backends := *backendsArrPtr

	ln := len(backends)
	rand.Seed(time.Now().UTC().UnixNano())
	index := rand.Intn(ln)

	var chosenIndex int
	ind := index
	maxCred := int32(0)
	backend2Return := backends[ind]
	for {
		log.Println("MostCredits: Looking at:", backends[ind].Ip, "has credits =", backends[ind].Credits) // debug
		if maxCred < backends[ind].Credits {
			maxCred = backends[ind].Credits
			backend2Return = backends[ind]
			chosenIndex = ind
		}
		ind = (ind + 1) % ln
		if ind == index {
			break
		}
	}

	log.Println("MostCredits:", chosenIndex, ", credits:", maxCred)
	return backend2Return, nil
}
