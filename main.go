package main

import (
	"log"
	"net/http"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/internal/globals"
	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	irequest "github.com/MSrvComm/MiCoProxy/internal/request"
	"github.com/gorilla/mux"
)

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT
	inProxy := incoming.NewProxy(globals.RedirectUrl_g)

	send := make(chan *irequest.Request, 10)
	outProxy := outgoing.NewRequestHandler(send)
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outProxy.HandleOutgoing)

	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(inProxy.Handle)

	lb := loadbalancer.NewLBThread(send)
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
