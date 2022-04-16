package inServer

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/request"
)

type InServer struct {
	rw    *sync.Mutex
	queue []*request.Request
	proxy *incoming.Proxy
}

func NewInServer(target string) *InServer {
	proxy := incoming.NewProxy(target)
	return &InServer{rw: &sync.Mutex{}, proxy: proxy}
}

func (inSrv *InServer) Len() int { return len(inSrv.queue) }

func (inSrv *InServer) Less(i, j int) bool {
	// We want Pop to give us the lowest, not highest, priority so we use lesser than here.
	return inSrv.queue[i].Priority < inSrv.queue[j].Priority
}

func (inSrv *InServer) Swap(i, j int) {
	inSrv.queue[i], inSrv.queue[j] = inSrv.queue[j], inSrv.queue[i]
	inSrv.queue[i].Index = i
	inSrv.queue[j].Index = j
}

func (inSrv *InServer) Push(r any) {
	// inSrv.rw.Lock()
	// defer inSrv.rw.Unlock()
	n := len(inSrv.queue)
	rq := r.(*request.Request)
	rq.Index = n
	inSrv.queue = append(inSrv.queue, rq)
	// heap.Fix(inSrv, rq.Index)
}

func (inSrv *InServer) Pop() any {
	if inSrv.Len() == 0 {
		return nil
	}
	ind := 0
	minTime, _ := inSrv.queue[0].Rq.Context().Deadline()
	for i := range inSrv.queue {
		if dl, _ := inSrv.queue[i].Rq.Context().Deadline(); !dl.After(minTime) {
			minTime, _ = inSrv.queue[i].Rq.Context().Deadline()
			ind = i
		}
	}
	// request to return
	rq := inSrv.queue[ind]
	n := inSrv.Len()
	old := inSrv.queue
	// swap the last element in the list
	// with the element at the index
	tmp := old[n-1]
	old[n-1] = nil
	old[ind] = tmp
	// remove the element we are returning
	inSrv.queue = old[0 : n-1]
	return rq
}

// func (inSrv *InServer) Pop() any {
// 	inSrv.rw.Lock()
// 	defer inSrv.rw.Unlock()
// 	if inSrv.Len() == 0 {
// 		return nil
// 	}
// 	old := inSrv.queue
// 	n := inSrv.Len()
// 	item := old[n-1]
// 	old[n-1] = nil
// 	item.Index = -1
// 	inSrv.queue = old[0 : n-1]
// 	return item
// }

// func (inSrv *InServer) Update() {
// 	// inSrv.rw.Lock()
// 	// defer inSrv.rw.Unlock()
// 	for i := range inSrv.queue {
// 		rq := inSrv.queue[i]
// 		rq.Mu.Lock()
// 		rq.SetPriority()
// 		// heap.Fix(inSrv, rq.Index)
// 		rq.Mu.Unlock()
// 	}
// }

func (inSrv *InServer) IsEmpty() bool {
	return inSrv.Len() == 0
}

func (inSrv *InServer) Handle(w http.ResponseWriter, r *http.Request) {
	done := make(chan bool)
	rq := request.NewRequest(w, r, done)
	if time.Since(rq.StartTime) < time.Second {
		w.WriteHeader(http.StatusEarlyHints)
		fmt.Fprintf(w, "not enough time left")
	}
	inSrv.Push(rq)
	<-done // block the original request here
}

// func (inSrv *InServer) Run() {
// 	ticker := time.NewTicker(time.Millisecond * 200)
// 	for {
// 		select {
// 		case <-ticker.C:
// 			inSrv.Update()
// 		default:
// 			for {
// 				if inSrv.IsEmpty() {
// 					break
// 				}
// 				rq := inSrv.Pop().(*request.Request)
// 				go func() {
// 					inSrv.proxy.Handle(rq.Wr, rq.Rq)
// 					rq.Done <- true // release the original handler
// 				}()
// 			}
// 		}
// 	}
// }

func (inSrv *InServer) Run() {
	for {
		if inSrv.IsEmpty() {
			continue
		}
		rq := inSrv.Pop().(*request.Request)
		go func() {
			inSrv.proxy.Handle(rq.Wr, rq.Rq)
			rq.Done <- true // release the original handler
		}()
	}
}
