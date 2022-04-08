package incoming

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

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
	target         *url.URL
	proxy          *httputil.ReverseProxy
	activeReqs     uint64
	capacity       uint64
	reserved       uint64 // number of requests expected to be on the way
	tickerInterval time.Duration
}

func NewProxy(target string, capacity uint64) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{target: url, proxy: httputil.NewSingleHostReverseProxy(url), activeReqs: 0,
		capacity: capacity, tickerInterval: time.Microsecond * 200}
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	// set forwarded for header
	log.Println("incoming") // debug
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	p.activeReqs++
	w.Header().Set("X-Forwarded-For", s)

	p.proxy.Transport = &pTransport{}
	p.proxy.ServeHTTP(w, r)
	p.activeReqs--
}

func (p *Proxy) reservedTimeout() {
	ticker := time.NewTicker(time.Microsecond * 200)
	select {
	case <-ticker.C:
		p.reserved = 0
	}
}

func (p *Proxy) Probe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	bw := p.capacity - p.activeReqs - p.reserved
	bw = div(bw) // return 50% of current capacity
	w.Header().Set("BANDWIDTH", fmt.Sprintf("%d", bw))
}

type number interface {
	int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64
}

func div[T number](a T) T {
	if a == 0 {
		return a
	}
	if 1 <= a && a <= 3 {
		return 1
	}
	if a%2 != 0 {
		a -= 1
	}
	return a / 2
}
