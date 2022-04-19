package credits

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
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
	Reqs        int32 // active requests on the server, populated by InProxy
	sendInteval time.Duration
	upstream    string
	conf        *config.Config
	Frontends   []*frontend
}

func NewCreditProxy(conf *config.Config) *CreditProxy {
	interval, err := strconv.Atoi(os.Getenv("SEND_INTERVAL"))
	if err != nil {
		log.Fatal("valid interval required to send credits")
	}
	upstream := os.Getenv("UPSTREAM")
	return &CreditProxy{
		RW:          &sync.RWMutex{},
		Reqs:        0,
		sendInteval: time.Duration(interval) * time.Second,
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
		// backendsMap := *cp.conf.BackendMap[svc]
		for i := range cp.conf.BackendMap[svc] {
			// for i := range backendsMap {
			totalCredits += cp.conf.BackendMap[svc][i].Credits
			// totalCredits += backendsMap[i].Credits
		}
	}
	if totalCredits == 0 {
		totalCredits = int32(backends.InitCredits)
	}
	totalCredits -= cp.Reqs // adjust for the number of requests already in the system
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
	_ = atomic.SwapInt32(&backend.Credits, int32(credits))
	log.Println("Updating Backend Credits: Received fron", ip, "updating:", backend.Ip,
		"with:", credits, "has credits:", backend.Credits) // debug
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
	// early return if the node does not have any credit yet
	// or doesn't know of any downstream services
	if totalCredits == 0 {
		return
	}
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
	rand.Seed(time.Now().UTC().UnixNano())
	index := rand.Intn(ln)
	i := index
	for {
		if totalCredits < crDelta {
			log.Println("SendCreditMessage: Breaking out:", totalCredits)
			break
		}
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

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("SendCreditMessage:", err.Error())
			return
		}
		// update frontend's quota once request sent successfully
		cp.Frontends[i].credits = crDelta
		log.Println("SendCreditMessage: Response Status Code", resp.StatusCode)

		totalCredits -= crDelta

		i = (i + 1) % ln
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
