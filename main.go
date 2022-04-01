package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/loadbalancer"
	"github.com/gorilla/mux"
)

type myTransport struct{}

func (t *myTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Println("Error response in myTransport RoungTrip: ", err)
		return nil, err
	}

	_, err = httputil.DumpResponse(resp, true)
	if err != nil {
		log.Println("Error in dumping response in myTransport RoundTrip: ", err)
		return nil, err
	}
	return resp, nil
}

type Proxy struct {
	reqs   int
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func NewProxy(target string) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{target: url, proxy: httputil.NewSingleHostReverseProxy(url)}
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	// set forwarded for header
	log.Println("incoming") // used for counting incoming requests
	p.reqs++
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("X-Forwarded-For", s)

	p.proxy.Transport = &myTransport{}
	p.proxy.ServeHTTP(w, r)
	w.Header().Set("Reqs", fmt.Sprint(p.reqs))
	p.reqs--
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

func handleOutgoing(w http.ResponseWriter, r *http.Request) {
	r.URL.Scheme = "http"
	r.RequestURI = ""

	svc, port, err := net.SplitHostPort(r.Host)
	if err == nil {
		addService(svc)
	}

	backend, err := loadbalancer.NextEndpoint(svc)
	if err != nil {
		log.Println("Error fetching backend:", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	r.URL.Host = net.JoinHostPort(backend.Ip, port)   // use the ip directly
	globals.Svc2BackendSrvMap_g.Incr(svc, backend.Ip) // a new request

	client := &http.Client{Timeout: time.Second * 10}
	start := time.Now()
	resp, err := client.Do(r)

	globals.Svc2BackendSrvMap_g.Decr(svc, backend.Ip) // close the request
	elapsed := time.Since(start).Nanoseconds()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	// rqs, _ := strconv.Atoi(resp.Header.Get("Reqs"))

	loadbalancer.System_reqs_g++
	rtt := uint64(elapsed)
	loadbalancer.System_rtt_avg_g = uint64(float64(loadbalancer.System_rtt_avg_g) + (float64(rtt-loadbalancer.System_rtt_avg_g) / float64(loadbalancer.System_reqs_g)))

	backend.RW.Lock()
	defer backend.RW.Unlock()
	// backend.Reqs = int64(rqs) // update active request on the backend metadata
	backend.RcvTime = start
	backend.Count++
	delta := float64(float64(elapsed)-backend.WtAvgRTT) / float64(backend.Count)
	backend.WtAvgRTT += delta
	backend.NoSched = false // since we heard from this backend we can include it in scheduling decisions
}

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT
	fmt.Println("Input Port", globals.PROXYINPORT)
	fmt.Println("Output Port", globals.PROXOUTPORT)
	fmt.Println("redirecting to:", globals.RedirectUrl_g)
	fmt.Println("User ID:", os.Getuid())
	proxy := NewProxy(globals.RedirectUrl_g)
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(handleOutgoing)

	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(proxy.handle)

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	go func() { log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux)) }()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
