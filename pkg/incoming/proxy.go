package incoming

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"github.com/MSrvComm/MiCoProxy/pkg/credits"
	"github.com/gin-gonic/gin"
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

type InProxy struct {
	cp    *credits.CreditProxy
	proxy *httputil.ReverseProxy
}

func NewInProxy(target string, cp *credits.CreditProxy) *InProxy {
	url, _ := url.Parse(target)
	return &InProxy{cp: cp, proxy: httputil.NewSingleHostReverseProxy(url)}
}

func (p *InProxy) Handle(c *gin.Context) {
	log.Println("incoming")
	atomic.AddInt32(&p.cp.Reqs, 1) // add the active request to credit proxy too

	s, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		log.Println(err)
	}

	c.Writer.Header().Add("X-Forwarded-For", s)
	p.proxy.Transport = &pTransport{}
	p.proxy.ServeHTTP(c.Writer, c.Request)

	// c.Writer.Header().Set("CREDITS", "1") // return a credit with every response
	atomic.AddInt32(&p.cp.Reqs, -1) // remove the active request from credit proxy
	p.cp.AddFrontend(s)
}
