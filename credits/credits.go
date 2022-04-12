package credits

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
)

type creditProxy struct {
	frontends    []globals.BackendSrv
	backends     []globals.BackendSrv
	lastIndex    int
	capacity     float64
	creditUpdate chan bool
}

func NewCreditProxy(creditUpdate chan bool) *creditProxy {
	return &creditProxy{
		capacity:     incoming.Capacity_g,
		creditUpdate: creditUpdate,
		backends:     make([]globals.BackendSrv, 0),
		lastIndex:    0,
	}
}

func (cp *creditProxy) assignNewCredit(n int64) {
	cp.frontends = globals.Svc2BackendSrvMap_g.Get(globals.Upstream_svc_g)
	if len(cp.frontends) == 0 {
		log.Println("assignNewCredit: no frontends") // debug
		return
	}
	cp.frontends[cp.lastIndex].ChangeCredit(n)
	cp.sndCreditMsg(cp.lastIndex, n)
	cp.lastIndex = (cp.lastIndex + 1) % len(cp.frontends)
}

func (cp *creditProxy) Run() {
	for range cp.creditUpdate {
		var creditsTotal float64
		for i := range cp.backends {
			creditsTotal += float64(cp.backends[i].CreditsBackend)
		}
		if creditsTotal == 0 {
			creditsTotal = incoming.Capacity_g
		}
		log.Println("creditUpdate: total", creditsTotal) // debug
		credits := int64(creditsTotal / float64(len(cp.frontends)))
		if credits <= 0 {
			credits = 1
		}
		log.Println("creditUpdate: sending", credits) // debug
		cp.assignNewCredit(credits)
	}
}

func (cp *creditProxy) sndCreditMsg(index int, credit int64) {
	url := "http://" + cp.frontends[index].Ip + globals.CLIENTPORT + "/credits"
	log.Println("SendCreditMessage:", url) // debug
	body := []byte("")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Println("SendCreditMessage:", err.Error())
		return
	}
	req.Header.Set("CREDITS", fmt.Sprintf("%d", credit))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("SendCreditMessage:", err.Error())
		return
	}
	log.Println("SendCreditMessage: Response Status Code", resp.StatusCode)
}

func (cp *creditProxy) rcvCreditMsg(ip string, credits int) {
	backend := globals.Svc2BackendSrvMap_g.SearchByHostIP(ip)
	log.Println("rcvCreditMsg: Searching for", ip)
	if backend == nil {
		log.Println("rcvCreditMsg: nil backend")
		return
	}
	backend.CreditsBackend += int64(credits)
	log.Println("rcvCreditMsg: backend:", backend.Ip, "credit added:", backend.CreditsBackend)
	exists := false
	for i := range cp.backends {
		if backend == &cp.backends[i] {
			cp.backends[i] = *backend
			exists = true
		}
	}
	if !exists {
		cp.backends = append(cp.backends, *backend)
	}
}

func (cp *creditProxy) Listen(w http.ResponseWriter, r *http.Request) {
	log.Println("creditProxy Listen hit")
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	var credits int
	c := r.Header.Get("CREDITS")
	if c == "" {
		log.Println("Empty CREDITS received")
		credits = 0
	} else {
		credits, _ = strconv.Atoi(c)
		log.Println("CREDITS received:", credits)
	}
	cp.rcvCreditMsg(s, credits)
	w.WriteHeader(http.StatusAccepted)
}
