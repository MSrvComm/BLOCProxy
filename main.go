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
	"time"
)

const (
	CLIENTPORT   = ":5000"
	PROXYINPORT  = ":62081" // which port will the reverse proxy use for making outgoing request
	PROXYOUTPORT = ":62082" // which port the reverse proxy listens on
)

var (
	g_redirectUrl string
	globalMap     = make(map[string]PathStats) // used for timing
)

// used for timing
type PathStats struct {
	Path      string
	Count     int
	totalTime int64
	RTT       int64
	AvgRTT    int64
}

type myTransport struct{}

func (t *myTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Print("\n\ncame in error resp here", err)
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
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func NewProxy(target string) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{target: url, proxy: httputil.NewSingleHostReverseProxy(url)}
}

func (p *Proxy) handle(w http.ResponseWriter, r *http.Request) {
	// set forwarded for header
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("X-Forwarded-For", s)

	fmt.Println(r.URL.Path)
	p.proxy.Transport = &myTransport{}
	p.proxy.ServeHTTP(w, r)
}

func handleOutgoing(w http.ResponseWriter, r *http.Request) {

	// TODO: implement load balancing here

	key := r.Method + r.URL.Path // used for timing
	start := time.Now()          // used for timing
	r.URL.Scheme = "http"
	r.URL.Host = r.Host
	r.RequestURI = ""

	// // supporting http2
	// http2.ConfigureTransport(http.DefaultTransport.(*http.Transport))

	response, err := http.DefaultClient.Do(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	// // implementing a flusher
	// done := make(chan bool)
	// go func() {
	// 	for {
	// 		select {
	// 		case <-time.Tick(10 * time.Millisecond):
	// 			w.(http.Flusher).Flush()
	// 		case <-done:
	// 			return
	// 		}
	// 	}
	// }()

	// // supporting trailers
	// trailerKeys := []string{}
	// for key := range response.Trailer {
	// 	trailerKeys = append(trailerKeys, key)
	// }
	// w.Header().Set("Trailer", strings.Join(trailerKeys, ","))

	w.WriteHeader(response.StatusCode)
	io.Copy(w, response.Body)

	// // adding trailers
	// for key, values := range response.Trailer {
	// 	for _, value := range values {
	// 		w.Header().Set(key, value)
	// 	}
	// }

	// close(done)

	elapsed := time.Since(start) // used for timing
	if val, ok := globalMap[key]; ok {
		val.Count += 1
		val.RTT = elapsed.Nanoseconds()
		val.totalTime += val.RTT
		val.AvgRTT = val.totalTime / int64(val.Count)
	} else {
		var m PathStats
		m.Count = 1
		m.RTT = elapsed.Nanoseconds()
		val.totalTime = val.RTT
		val.AvgRTT = val.RTT
		globalMap[key] = m
	}
}

func getStats(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, globalMap)
}

func main() {
	g_redirectUrl = "http://localhost" + CLIENTPORT
	fmt.Println("Input Port", PROXYINPORT)
	fmt.Println("Output Port", PROXYOUTPORT)
	fmt.Println("redirecting to:", g_redirectUrl)
	fmt.Println("User ID:", os.Getuid())
	proxy := NewProxy(g_redirectUrl)
	outMux := http.NewServeMux()
	outMux.HandleFunc("/", handleOutgoing)
	inMux := http.NewServeMux()
	inMux.HandleFunc("/", proxy.handle)
	inMux.HandleFunc("/stats", getStats)
	go func() { log.Fatal(http.ListenAndServe(PROXYINPORT, inMux)) }()
	log.Fatal(http.ListenAndServe(PROXYOUTPORT, outMux))
}
