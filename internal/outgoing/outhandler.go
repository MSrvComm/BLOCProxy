package outgoing

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
)

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
	time.Sleep(time.Nanosecond * 100) // enough for the request handler not to hit the list before it's populated
}

func HandleOutgoing(w http.ResponseWriter, r *http.Request) {
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

	r.URL.Host = net.JoinHostPort(backend.Ip, port) // use the ip directly
	backend.Incr()                                  // a new request

	client := &http.Client{Timeout: time.Second * 10}
	start := time.Now()
	resp, err := client.Do(r)

	backend.Decr() // close the request

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	if resp.StatusCode != 200 {
		log.Println("Request was dropped") // debug
		backend.Backoff()                  // backoff from this backend for a while
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintf(w, "Bad reply from server")
	}

	// we always receive a new credit value from the backend
	// it can be a 1 or a 0
	credits, _ := strconv.Atoi(resp.Header.Get("CREDITS"))
	elapsed := time.Since(start).Nanoseconds()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	go backend.Update(start, int64(credits), uint64(elapsed))
}
