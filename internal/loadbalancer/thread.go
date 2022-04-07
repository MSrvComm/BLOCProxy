package loadbalancer

import (
	"log"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/queue"
	irequest "github.com/MSrvComm/MiCoProxy/internal/request"
)

type LBThread struct {
	queue *queue.Queue
	rcv   chan *irequest.Request
}

func NewLBThread(rcv chan *irequest.Request) *LBThread {
	return &LBThread{queue: queue.NewQueue(), rcv: rcv}
}

func (lb *LBThread) sendRequests() {
	for {
		if lb.queue.IsEmpty() {
			break
		}
		rq, err := lb.queue.Dequeue()
		if err != nil {
			log.Println("Error fetching request")
			continue
		}
		rq.MakeRequest("http://google.com")
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
