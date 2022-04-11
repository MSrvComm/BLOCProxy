package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	"github.com/gorilla/mux"
)

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT
	fmt.Println("Input Port", globals.PROXYINPORT)
	fmt.Println("Output Port", globals.PROXOUTPORT)
	fmt.Println("redirecting to:", globals.RedirectUrl_g)
	fmt.Println("User ID:", os.Getuid())

	// get capacity
	incoming.Capacity_g, _ = strconv.ParseFloat(os.Getenv("CAPACITY"), 64)
	// get upstream service(s)
	globals.Upstream_svc_g = os.Getenv("UPSTREAM")
	// get list of upstream of pods
	if globals.Upstream_svc_g != "" {
		if !loadbalancer.PopulateSvcList(globals.Upstream_svc_g) {
			log.Fatal("No pod for upstream found")
		}
	}

	// incoming request handling
	proxy := incoming.NewProxy(globals.RedirectUrl_g)
	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(proxy.Handle)

	// outgoing request handling
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outgoing.HandleOutgoing)

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	// start the proxy services
	go func() {
		log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux))
	}()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
