package incoming

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

var Capacity_g float64

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
	target       *url.URL
	proxy        *httputil.ReverseProxy
	activeReqs   int64
	creditUpdate chan bool
}

func NewProxy(target string, creditUpdate chan bool) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{
		target:       url,
		proxy:        httputil.NewSingleHostReverseProxy(url),
		activeReqs:   0,
		creditUpdate: creditUpdate,
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

	// if there are too many requests then ask the client to retry
	if p.count()+1 > int64(Capacity_g) {
		// log.Println(p.activeReqs, Capacity_g, "Sending Early Hints")
		log.Println(p.activeReqs, Capacity_g, "Rejecting Request")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	p.add(1)
	log.Println("accepted")
	w.Header().Set("X-Forwarded-For", s)

	p.proxy.Transport = &pTransport{}
	p.proxy.ServeHTTP(w, r)

	// we can send a 0 or a 1 credit back
	// if the backend receives 0, they can't send another request for a second
	// the probability of a credit being sent is based on how loaded the system is right now
	// capacity_g hard codes the capacity of the system for the moment
	var credits string
	if rand.Float64() < float64(p.count())/Capacity_g {
		credits = "0"
	} else {
		credits = "1"
	}

	w.Header().Set("CREDITS", credits)
	log.Println("Active Requests:", p.activeReqs, ", credits:", credits)
	p.add(-1)
	p.creditUpdate <- true // let the credit system know
}
