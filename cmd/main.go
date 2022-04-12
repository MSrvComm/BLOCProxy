package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/credits"
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

	// get downstream service(s)
	globals.Downstream_svc_g = os.Getenv("DOWNSTREAM")
	// get list of downstream of pods
	if globals.Downstream_svc_g != "" {
		if !loadbalancer.PopulateSvcList(globals.Downstream_svc_g) {
			log.Println("No pod for downstream found")
		}
	}

	// incoming request handling
	creditUpdate := make(chan bool, 10) // used to update the credit system that another response is sent
	globals.InProxy = incoming.NewProxy(globals.RedirectUrl_g, creditUpdate)
	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(globals.InProxy.Handle)

	// outgoing request handling
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outgoing.HandleOutgoing)

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	// start the credit system
	cp := credits.NewCreditProxy(creditUpdate)
	go func() {
		cp.Run()
	}()

	// start the proxy services
	go func() {
		log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux))
	}()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
