package incoming

import (
	"fmt"
	"log"

	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/logger"
)

// var (
// 	// Capacity_g int64
// 	// SLO        float64
// 	RunAvg_g  = true // average has not been set in env
// 	Start_g   = time.Now()
// 	timeSet_g = false
// 	count     = 0
// 	avg_g     = float64(0)
// )

type pTransport struct{}

func (t *pTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Print("\n\ncame in error resp here: ", err)
		return nil, err
	}

	_, err = httputil.DumpResponse(response, true) // check if the response is valid
	if err != nil {
		log.Print("\n\nerror in dump response\n")
		return nil, err
	}
	return response, err
}

type Proxy struct {
	target     *url.URL
	proxy      *httputil.ReverseProxy
	activeReqs int64
	data       chan logger.Data
	schan      chan bool
	qchan      chan int64
	dchan      chan bool
}

func NewProxy(target string, data chan logger.Data, schan, dchan chan bool, qchan chan int64) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{
		target:     url,
		proxy:      httputil.NewSingleHostReverseProxy(url),
		activeReqs: 0,
		data:       data,
		schan:      schan,
		qchan:      qchan,
		dchan:      dchan,
	}
}

func (p *Proxy) add(n int64) {
	atomic.AddInt64(&p.activeReqs, n)
}

func (p *Proxy) count() int64 {
	return atomic.LoadInt64(&p.activeReqs)
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	log.Println("incoming")
	// if !timeSet_g {
	// 	timeSet_g = true
	// 	Start_g = time.Now()
	// }

	// // avg_g = alpha_g*avg_g + (1-alpha_g)*float64(p.count())
	// count++
	// avg_g = avg_g + (float64((p.count()+1))-avg_g)/float64(count)
	// log.Println("Avg is:", avg_g) // debug
	// if loadbalancer.DefaultLBPolicy_g == "MLeastConn" && RunAvg_g && time.Since(Start_g) > 30*time.Second {
	// 	// globals.Capacity_g = uint64(math.Ceil(avg_g))
	// 	// log.Println("Setting Capacity to:", avg_g) // debug
	// 	// reset all counters
	// 	log.Println("Resetting counters") // debug
	// 	Start_g = time.Now()
	// 	count = 0
	// 	avg_g = 0
	// }

	// not checking admission if capacity not set or loadbalancer is not "MLeastConn"
	if loadbalancer.DefaultLBPolicy_g == "MLeastConn" && globals.Capacity_g != 0 {
		// if there are too many requests then ask the client to retry
		if uint64(p.count()+1) > globals.Capacity_g {
			// log.Println(p.activeReqs, Capacity_g, "Sending Early Hints")
			log.Println(p.activeReqs, globals.Capacity_g, "Rejecting Request")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, "Retry")
			return
		}
	}
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	p.add(1)
	go func() { p.schan <- true }()      // queueing theory
	go func() { p.qchan <- p.count() }() // queueing theory
	log.Println("accepted")
	w.Header().Set("X-Forwarded-For", s)

	p.proxy.Transport = &pTransport{}
	start := time.Now() // machine learning
	p.proxy.ServeHTTP(w, r)
	elapsed := time.Since(start) // machine learning
	// go func() { p.data <- logger.Data{Count: p.count(), Elapsed: elapsed} }() // machine learning
	msg := fmt.Sprintf("timing: elapsed: %v, count: %d", elapsed, p.count())
	log.Println(msg) // debug

	// we can send a 0 or a 1 credit back
	// if the backend receives 0, they can't send another request for a second
	// the probability of a credit being sent is based on how loaded the system is right now
	// capacity_g hard codes the capacity of the system for the moment
	var chip string
	if rand.Float64() < float64(p.count())/(0.8*float64(globals.Capacity_g)) {
		chip = "0"
	} else {
		chip = "1"
	}

	w.Header().Set("CHIP", chip)
	p.add(-1)
	go func() { p.dchan <- true }() // queueing theory
}
