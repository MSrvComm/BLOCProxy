package incoming

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	target *url.URL
	proxy  *httputil.ReverseProxy
}

func NewProxy(target string) *Proxy {
	url, _ := url.Parse(target)
	return &Proxy{target: url, proxy: httputil.NewSingleHostReverseProxy(url)}
}

func (p *Proxy) Handle(w http.ResponseWriter, r *http.Request) {
	// set forwarded for header
	log.Println("incoming") // debug
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("X-Forwarded-For", s)

	p.proxy.Transport = &pTransport{}
	p.proxy.ServeHTTP(w, r)
}
