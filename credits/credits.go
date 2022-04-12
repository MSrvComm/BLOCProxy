package credits

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/gorilla/mux"
)

type creditProxy struct {
	creditsStarted bool
	frontends      []globals.BackendSrv
	backends       []globals.BackendSrv
	ln             int
	lastIndex      int
	capacity       float64
	creditUpdate   chan bool
}

func NewCreditProxy(creditUpdate chan bool) *creditProxy {
	return &creditProxy{
		creditsStarted: false,
		capacity:       incoming.Capacity_g,
		creditUpdate:   creditUpdate,
		backends:       make([]globals.BackendSrv, 0),
	}
}

func (cp *creditProxy) genCreditsFirstTime(svc string) {
	frontends := globals.Svc2BackendSrvMap_g.Get(svc)
	log.Println("genCreditsFirstTime: frontends", frontends)
	if frontends == nil {
		return
	}
	ln := len(frontends)
	index := rand.Intn(ln)
	i := index + 1
	capa := cp.capacity
	for ; capa != 0; i = (i + 1) % ln {
		frontends[i].ChangeCredit(1)
		capa--
	}
	cp.frontends = frontends
	cp.lastIndex = i
	cp.ln = ln
	cp.creditsStarted = true
}

func (cp *creditProxy) assignNewCredit(svc string, n int64) {
	if globals.Downstream_svc_g == "" {
		return // nothing to assign credit to
	}
	if cp.creditsStarted {
		cp.genCreditsFirstTime(svc)
		return
	}
	cp.frontends[cp.lastIndex].ChangeCredit(n)
	cp.sndCreditMsg(cp.lastIndex, 1)
	cp.lastIndex = (cp.lastIndex + 1) % cp.ln
}

func (cp *creditProxy) Run() {
	go func() {
		creditMux := mux.NewRouter()
		creditMux.PathPrefix("/").HandlerFunc(cp.Listen)
		log.Fatal(http.ListenAndServe(globals.CREDIPORT, creditMux))
	}()
	if !cp.creditsStarted {
		cp.genCreditsFirstTime(globals.Downstream_svc_g)
	}
	for range cp.creditUpdate {
		cp.assignNewCredit(globals.Downstream_svc_g, 1)
	}
}

func (cp *creditProxy) sndCreditMsg(index, credit int) {
	url := "http://" + cp.frontends[index].Ip + globals.CREDIPORT
	log.Println("SendCreditMessage:", url) // debug
	body := []byte("")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Println("SendCreditMessage:", err.Error())
		return
	}
	req.Header.Set("CREDIT", fmt.Sprintf("%d", credit))
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
	backend.CreditsBackend = int64(credits)
	exists := false
	for i := range cp.backends {
		if backend == &cp.backends[i] {
			exists = true
		}
	}
	if !exists {
		cp.backends = append(cp.backends, *backend)
	}
}

func (cp *creditProxy) Listen(w http.ResponseWriter, r *http.Request) {
	s, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Println(err)
	}
	var credits int
	c := r.Header.Get("CREDITS")
	if c == "" {
		credits = 0
	} else {
		credits, _ = strconv.Atoi(c)
	}
	cp.rcvCreditMsg(s, credits)
	w.WriteHeader(http.StatusAccepted)
}
