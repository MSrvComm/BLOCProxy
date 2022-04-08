package loadbalancer

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/MSrvComm/MiCoProxy/internal/globals"
)

type Request struct {
	d chan bool
	r *http.Request
	w http.ResponseWriter
}

func NewRequest(w http.ResponseWriter, r *http.Request, d chan bool) *Request {
	return &Request{w: w, r: r, d: d}
}

func (rq *Request) makeRequest(svc, port string, backend *globals.BackendSrv) {
	// release the handler function HandleOutgoing
	// no matter what!
	defer func() { rq.d <- true }()

	r := rq.r
	w := rq.w

	r.URL.Scheme = "http"
	r.RequestURI = ""

	r.URL.Host = net.JoinHostPort(backend.Ip, port) // use the ip directly

	globals.Svc2BackendSrvMap_g.Incr(svc, backend.Ip) // a new request

	client := &http.Client{Timeout: time.Second * 10}
	start := time.Now()
	resp, err := client.Do(r)                         // making the request
	globals.Svc2BackendSrvMap_g.Decr(svc, backend.Ip) // close the request
	elapsed := time.Since(start).Nanoseconds()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	log.Println(resp)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	// rq.d <- true // release the handler function HandleOutgoing here explicitly
	// backend.RW.Lock()
	// defer backend.RW.Unlock()
	// backend.RcvTime = start
	// backend.LastRTT = uint64(elapsed)

	// we want to avoid paying the locking cost
	// for stats update in the call
	go rq.statsUpdate(backend, start, uint64(elapsed))
} // the request released here

func (rq *Request) statsUpdate(backend *globals.BackendSrv, start time.Time, elapsed uint64) {
	backend.RW.Lock()
	defer backend.RW.Unlock()
	backend.RcvTime = start
	backend.LastRTT = elapsed
}
