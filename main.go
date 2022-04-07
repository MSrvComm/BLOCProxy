package main

import (
	"log"
	"net/http"

	"github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	irequest "github.com/MSrvComm/MiCoProxy/internal/request"
	"github.com/gorilla/mux"
)

func main() {
	inProxy := incoming.NewProxy("google.com")

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

	go func() {
		log.Fatal(http.ListenAndServe(":8080", outMux))
	}()
	log.Fatal(http.ListenAndServe(":8081", inMux))
}
