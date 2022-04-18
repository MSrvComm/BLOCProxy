package credits

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MSrvComm/MiCoProxy/pkg/backends"
	"github.com/MSrvComm/MiCoProxy/pkg/config"
	"github.com/gin-gonic/gin"
)

type frontend struct {
	ip      string
	credits int32
}

func newFrontEnd(ip string) *frontend {
	return &frontend{ip: ip}
}

type CreditProxy struct {
	RW          *sync.RWMutex
	sendInteval time.Duration
	upstream    string
	conf        *config.Config
	Frontends   []*frontend
}

func NewCreditProxy(conf *config.Config) *CreditProxy {
	// interval, err := strconv.Atoi(os.Getenv("SEND_INTERVAL"))
	// if err != nil {
	// 	log.Fatal("valid interval required to send credits")
	// }
	upstream := os.Getenv("UPSTREAM")
	return &CreditProxy{
		RW:          &sync.RWMutex{},
		sendInteval: 2 * time.Second,
		upstream:    upstream,
		conf:        conf,
		Frontends:   make([]*frontend, 0),
	}
}

func (cp *CreditProxy) AddFrontend(ip string) {
	cp.RW.RLock()
	for i := range cp.Frontends {
		if cp.Frontends[i].ip == ip {
			return
		}
	}
	cp.RW.RUnlock()
	fe := newFrontEnd(ip)
	cp.RW.Lock()
	cp.Frontends = append(cp.Frontends, fe)
	cp.RW.Unlock()
}

func (cp *CreditProxy) calculateCredits() int32 {
	totalCredits := int32(0)
	for svc := range cp.conf.BackendMap {
		for i := range cp.conf.BackendMap[svc] {
			totalCredits += cp.conf.BackendMap[svc][i].Credits
		}
	}
	if totalCredits == 0 {
		totalCredits = int32(backends.InitCredits)
	}
	return totalCredits
}

func (cp *CreditProxy) updateBackend(ip, cr string) {
	backend, err := cp.conf.ContainsSrv(ip)
	if err != nil {
		log.Println("updateBackend:", err)
		return
	}

	credits, err := strconv.Atoi(cr)
	if err != nil {
		log.Println("updateBackend: converting credits", err)
		return
	}
	log.Println("Credit Recieved:", credits)
	backend.Credits = atomic.SwapInt32(&backend.Credits, int32(credits))
}

func (cp *CreditProxy) Handle(c *gin.Context) {
	log.Println("creditProxy Listen hit")
	s, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		log.Println(err)
	}

	cr := c.Request.Header.Get("CREDITS")
	if cr == "" {
		return
	}
	cp.updateBackend(s, cr)
}

func (cp *CreditProxy) sendCredits() {
	if cp.upstream == "" {
		return
	}
	cport := fmt.Sprintf(":%d", cp.conf.ClientPort)
	totalCredits := cp.calculateCredits()
	ln := len(cp.Frontends)
	if ln == 0 {
		return
	}
	cr := totalCredits / int32(ln)

	// each frontend receives crDelta credits
	// but if there are not enough then each frontend receives
	// 1 credit till we run out of credits
	var crDelta int32
	if cr < 1 {
		crDelta = 1
	} else {
		crDelta = cr
	}
	for i := range cp.Frontends {
		url := "http://" + cp.Frontends[i].ip + cport + "/credits"
		log.Println("SendCreditMessage:", url) // debug
		body := []byte("")
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			log.Println("SendCreditMessage:", err.Error())
			return
		}
		log.Println("SendCreditMessage: credits:", crDelta) // debug
		req.Header.Set("CREDITS", fmt.Sprintf("%d", crDelta))
		totalCredits -= crDelta
		if totalCredits <= 0 {
			crDelta = 0
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("SendCreditMessage:", err.Error())
			return
		}
		log.Println("SendCreditMessage: Response Status Code", resp.StatusCode)
	}
}

func (cp *CreditProxy) Run(done chan bool) {
	ticker := time.NewTicker(cp.sendInteval)
	for {
		select {
		case <-ticker.C:
			cp.sendCredits()
		case <-done:
			return
		}
	}
}
