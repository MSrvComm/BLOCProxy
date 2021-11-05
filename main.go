package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

const (
	CLIENTPORT   = ":5000"
	PROXYINPORT  = ":62081" // which port will the reverse proxy use for making outgoing request
	PROXYOUTPORT = ":62082" // which port the reverse proxy listens on
)

var (
	g_redirectUrl string
)

type myTransport struct{}

func (t *myTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Print("\n\ncame in error resp here", err)
		return nil, err
	}
	_, err = httputil.DumpResponse(response, true)
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
	fmt.Println(r.URL.Path)
	fmt.Fprintf(w, "new incoming request on: %s\n", r.Host)
	p.proxy.Transport = &myTransport{}
	p.proxy.ServeHTTP(w, r)
}

func handleOutgoing(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "new outgoing request on: %s\n", r.Host)
	r.URL.Scheme = "https"
	r.URL.Host = r.Host
	response, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Println(err)
	}
	fmt.Fprint(w, response)
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
	go func() { log.Fatal(http.ListenAndServe(PROXYINPORT, inMux)) }()
	log.Fatal(http.ListenAndServe(PROXYOUTPORT, outMux))
}
