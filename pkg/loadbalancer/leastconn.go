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

// func (lb *LoadBalancer) MLeastConn(svc string) (*backends.Backend, error) {
// 	log.Println("Modified Least Connection used") // debug
// 	backends, err := lb.GetSvcList(svc)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// P2C Least Conn
// 	seed := time.Now().UTC().UnixNano()
// 	rand.Seed(seed)
// 	ln := len(backends)

// 	// we select two servers only if they have credit left
// 	index1 := rand.Intn(ln)

// 	for {
// 		if backends[index1].CreditsBackend > 0 {
// 			break
// 		}
// 		index1 = rand.Intn(ln)
// 	}

// 	index2 := rand.Intn(ln)

// 	for {
// 		if backends[index2].CreditsBackend > 0 {
// 			break
// 		}
// 		index2 = rand.Intn(ln)
// 	}

// 	srv1 := &backends[index1]
// 	srv2 := &backends[index2]

// 	var backend2Return *backends.Backend
// 	// var ip string
// 	if srv1.Reqs < srv2.Reqs {
// 		backend2Return = srv1
// 	} else {
// 		backend2Return = srv2
// 	}

// 	return backend2Return, nil
// }
