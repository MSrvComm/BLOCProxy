package loadbalancer

import (
	"log"
	"math/rand"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

func LeastConn(svc string) (*globals.BackendSrv, error) {
	log.Println("Least Connection used") // debug
	backends, err := GetSvcList(svc)
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
		return srv1, nil
	}
	return srv2, nil
}

func MLeastConn(svc string) (*globals.BackendSrv, error) {
	log.Println("Modified Least Connection used") // debug
	backends, err := GetSvcList(svc)
	if err != nil {
		return nil, err
	}

	// P2C Least Conn
	seed := time.Now().UTC().UnixNano()
	rand.Seed(seed)
	ln := len(backends)

	// we select two servers only if they have credit left
	index1 := rand.Intn(ln)

	for {
		if backends[index1].CreditsBackend > 0 {
			break
		}
		index1 = rand.Intn(ln)
	}

	index2 := rand.Intn(ln)

	for {
		if backends[index2].CreditsBackend > 0 {
			break
		}
		index2 = rand.Intn(ln)
	}

	srv1 := &backends[index1]
	srv2 := &backends[index2]

	var backend2Return *globals.BackendSrv
	// var ip string
	if srv1.Reqs < srv2.Reqs {
		backend2Return = srv1
	} else {
		backend2Return = srv2
	}

	return backend2Return, nil
}

func MLeastConnFull(svc string) (*globals.BackendSrv, error) {
	log.Println("Modified Full Least Connection used") // debug
	backends, err := GetSvcList(svc)
	if err != nil {
		return nil, err
	}

	ln := len(backends)

	// we select two servers if they have a credit
	// or it has been more than a second since the last response
	index := rand.Intn(ln)

	var backend2Return *globals.BackendSrv
	var minReqs int64

	log.Println("Backends length in MFullLC:", ln) // debug

	// i := (index + 1) % ln

	// for ; i != index; i = (i + 1) % ln {
	for i := 0; i < ln; i++ {
		log.Println("Backends in MFullLC:", backends[i].Ip, "has credit:", backends[i].CreditsBackend)
		if backends[i].CreditsBackend <= 0 {
			continue
		}
		if i == (index+1)%ln {
			backend2Return = &backends[i]
			minReqs = backends[i].Reqs
		}
		if backends[i].Reqs < minReqs {
			backend2Return = &backends[i]
			minReqs = backend2Return.Reqs
		}
	}
	log.Println("Backend2Return in MFullLC:", backend2Return)
	return backend2Return, nil
}
