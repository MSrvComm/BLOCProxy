package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/internal/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	"github.com/gorilla/mux"
)

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT

	int, _ := strconv.Atoi(os.Getenv("INTERVAL"))
	interval := time.Duration(int)

	cap, _ := strconv.Atoi(os.Getenv("CAPACITY"))

	inProxy := incoming.NewProxy(globals.RedirectUrl_g, uint64(cap))

	send := make(chan *loadbalancer.Request, 10)
	outProxy := outgoing.NewRequestHandler(send)
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outProxy.HandleOutgoing)

	inMux := mux.NewRouter()
	inMuxS := inMux.PathPrefix("/").Subrouter()
	inMuxS.HandleFunc("/", inProxy.Handle)
	inMuxS.HandleFunc("/probe", inProxy.Probe)

	lb := loadbalancer.NewLBThread(send, interval)
	defer lb.Close()
	go func() {
		lb.Run()
	}()

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	go func() {
		log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux))
	}()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
