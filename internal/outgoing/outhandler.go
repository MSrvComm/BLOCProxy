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

	// // first attempt
	// resp, err := callService(svc, port, w, r)

	// // retry 5 times if there are no errors
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	return
	// }
	// if resp.StatusCode == 103 {
	// 	for i := 0; i < 1; i++ {
	// 		resp, err = callService(svc, port, w, r)
	// 		if err != nil {
	// 			w.WriteHeader(http.StatusInternalServerError)
	// 			return
	// 		}
	// 		if resp.StatusCode != 103 {
	// 			break
	// 		}
	// 	}
	// }

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
	// we always receive a new credit value from the backend
	// it can be a 1 or a 0
	credits, _ := strconv.Atoi(resp.Header.Get("CREDITS"))
	elapsed := time.Since(start).Nanoseconds()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}

	// code := resp.StatusCode - 200
	// if code < 0 || code > 99 {
	if resp.StatusCode != 200 {
		log.Println("Request was dropped") // debug
		w.WriteHeader(resp.StatusCode)
		fmt.Fprintf(w, "Bad reply from server")
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Set(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	go backend.Update(start, uint64(credits), uint64(elapsed))
}

// func callService(svc, port string, w http.ResponseWriter, r *http.Request) (*http.Response, error) {
// 	log.Println("callService")
// 	backend, err := loadbalancer.NextEndpoint(svc)
// 	if err != nil {
// 		log.Println("Error fetching backend:", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		fmt.Fprint(w, err.Error())
// 		return nil, errors.New("invalid backend")
// 	}

// 	r.URL.Host = net.JoinHostPort(backend.Ip, port) // use the ip directly
// 	backend.Incr()                                  // a new request

// 	client := &http.Client{Timeout: time.Second * 10}
// 	start := time.Now()
// 	resp, err := client.Do(r)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var elapsed uint64
// 	if resp.StatusCode != 103 {
// 		// we always receive a new credit value from the backend
// 		// it can be a 1 or a 0
// 		credits, _ := strconv.Atoi(resp.Header.Get("CREDITS"))
// 		elapsed = uint64(time.Since(start).Nanoseconds())
// 		// we update the backend in a go routine
// 		// freeing up the handler function to return
// 		// separating the cost of the update from the cost of the service
// 		go backend.Update(start, elapsed, uint64(credits))
// 	}
// 	backend.Decr() // close the request
// 	return resp, nil
// }
