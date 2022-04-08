package loadbalancer

import (
	"log"
	"net"
	"strings"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/globals"
)

type LBThread struct {
	queue    *Queue
	rcv      chan *Request
	interval time.Duration
}

func NewLBThread(rcv chan *Request, interval time.Duration) *LBThread {
	return &LBThread{queue: NewQueue(), rcv: rcv, interval: interval}
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

			backend, _ = NextEndpoint(svc)
		}

		if rq != nil {
			if backend != nil {
				go rq.makeRequest(svc, port, backend)
			} else {
				log.Println("LB sendRequests: Nil backend")
				rq.d <- true // unblock the request
			}
		} else {
			log.Println("LB sendRequests: Nil Request")
		}
	}
}

func (lb *LBThread) Run() {
	ticker := time.NewTicker(lb.interval)
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
	time.Sleep(time.Millisecond * 50)
}
