package loadbalancer

import (
	"log"
	"math/rand"
	"sync"

	"github.com/MSrvComm/MiCoProxy/globals"
)

var lastSelections sync.Map

func RoundRobin(svc string) (*globals.BackendSrv, error) {
	log.Println("Round Robin used") // debug

	backends, err := GetBackendSvcList(svc)
	if err != nil {
		return nil, err
	}

	l := len(backends)

	ind, ok := lastSelections.Load(svc)
	var index int
	if !ok {
		index = rand.Intn(l)
	} else {
		index = ind.(int)
	}

	backend := &backends[index]
	index++
	index = index % l
	lastSelections.Store(svc, index)
	return backend, nil
}
