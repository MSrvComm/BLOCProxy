package loadbalancer

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/globals"
)

type LBThread struct {
	queue *Queue
	rcv   chan *Request
}

func NewLBThread(rcv chan *Request) *LBThread {
	return &LBThread{queue: NewQueue(), rcv: rcv}
}

func (lb *LBThread) sendRequests() {
	var backend *globals.BackendSrv
	for {
		if lb.queue.IsEmpty() {
			break
		}
		rq, err := lb.queue.Dequeue()
		if err != nil {
			log.Println("Error fetching request")
			continue
		}

		var svc string
		var port string
		if backend == nil {
			svc, port, err = net.SplitHostPort(rq.r.Host)
			if err == nil {
				addService(svc)
			}

			backend, err = NextEndpoint(svc)
		}

		go rq.MakeRequest(svc, port, backend)
	}
}

func (lb *LBThread) Run() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case r := <-lb.rcv:
			lb.queue.Enqueue(r)
		case <-ticker.C:
			lb.sendRequests()
		}
	}
}

func (lb *LBThread) Close() {
	close(lb.rcv)
}

func addService(s string) {
	// add the service we are looking for to the list of services
	// assumes we only ever make requests to internal servers

	// if the request is being made to epwatcher then it will create an infinite loop
	// we have also set a rule that any request to port 30000 is to be ignored
	if strings.Contains(s, "epwatcher") {
		return
	}

	for _, svc := range globals.SvcList_g {
		if svc == s {
			return
		}
	}
	globals.SvcList_g = append(globals.SvcList_g, s)
}
