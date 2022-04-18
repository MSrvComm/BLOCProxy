package outgoing

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/MSrvComm/MiCoProxy/pkg/config"
	"github.com/MSrvComm/MiCoProxy/pkg/loadbalancer"
	"github.com/gin-gonic/gin"
)

type OutProxy struct {
	conf *config.Config
	lb   *loadbalancer.LoadBalancer
}

func NewOutProxy(conf *config.Config) *OutProxy {
	return &OutProxy{conf: conf, lb: loadbalancer.NewLoadBalancer(conf)}
}

func (p *OutProxy) Handle(c *gin.Context) {
	c.Request.URL.Scheme = "http"
	c.Request.RequestURI = ""

	svc, port, err := net.SplitHostPort(c.Request.Host)
	if err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	backend, err := p.lb.NextEndpoint(svc)

	if err != nil {
		log.Println("Error fetching backend:", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(c.Writer, err.Error())
		return
	}

	c.Request.URL.Host = net.JoinHostPort(backend.Ip, port)

	client := &http.Client{Timeout: time.Second * 10}
	atomic.AddInt32(&backend.Reqs, 1)
	atomic.AddInt32(&backend.Credits, -1) // reduce a credit whenever request is made
	resp, err := client.Do(c.Request)
	atomic.AddInt32(&backend.Reqs, -1)
	// atomic.AddInt32(&backend.Credits, 1) // increase a credit whenever response is received

	if err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(c.Writer, err.Error())
		return
	}

	c.Writer.WriteHeader(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}
