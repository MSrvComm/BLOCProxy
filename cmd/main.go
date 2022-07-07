package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/logger"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	"github.com/gorilla/mux"
)

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT
	fmt.Println("Input Port", globals.PROXYINPORT)
	fmt.Println("Output Port", globals.PROXOUTPORT)
	fmt.Println("redirecting to:", globals.RedirectUrl_g)
	fmt.Println("User ID:", os.Getuid())

	loadbalancer.DefaultLBPolicy_g = os.Getenv("LBPolicy")
	if loadbalancer.DefaultLBPolicy_g == "MLeastConn" {
		globals.NumRetries_g, _ = strconv.Atoi(os.Getenv("RETRIES"))
		// get capacity
		capa, err := strconv.ParseInt(os.Getenv("CAPACITY"), 10, 64)
		if err != nil {
			globals.Capacity_g = 0.0
		} else {
			globals.Capacity_g = uint64(capa)
			globals.CapacityDefined = true
		}
		// var err error
		globals.SLO, err = strconv.ParseFloat(os.Getenv("SLO"), 64)
		if err != nil {
			globals.SLO = 0.0
		}
	} else {
		globals.NumRetries_g = 1
		globals.SLO = 1.0
		// globals.Capacity_g = 0
	}
	reset, _ := strconv.Atoi(os.Getenv("RESET"))
	globals.ResetInterval_g = time.Duration(reset) * time.Microsecond

	// // capacity has been set in the env; do not reset
	// if globals.Capacity_g != 0 {
	// 	incoming.RunAvg_g = false
	// }

	data := make(chan logger.Data, 100)
	qchan := make(chan int64, 100)
	schan := make(chan bool, 100)
	dchan := make(chan bool, 100)

	// incoming request handling
	proxy := incoming.NewProxy(globals.RedirectUrl_g, data, schan, dchan, qchan)
	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(proxy.Handle)

	// outgoing request handling
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outgoing.HandleOutgoing)

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	// Machine Learning Thread
	// mlt := logger.NewML(data)
	// go mlt.Run()

	// Queuing Theory Thread
	qtt := logger.NewQT(schan, dchan, qchan)
	go qtt.Run()

	// start the proxy services
	go func() {
		log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux))
	}()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
