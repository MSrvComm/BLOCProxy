package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/MSrvComm/MiCoProxy/controllercomm"
	"github.com/MSrvComm/MiCoProxy/globals"

	// "github.com/MSrvComm/MiCoProxy/internal/incoming"
	"github.com/MSrvComm/MiCoProxy/internal/inServer"
	"github.com/MSrvComm/MiCoProxy/internal/outgoing"
	"github.com/gorilla/mux"
)

func main() {
	globals.RedirectUrl_g = "http://localhost" + globals.CLIENTPORT
	fmt.Println("Input Port", globals.PROXYINPORT)
	fmt.Println("Output Port", globals.PROXOUTPORT)
	fmt.Println("redirecting to:", globals.RedirectUrl_g)
	fmt.Println("User ID:", os.Getuid())

	// globals.NumRetries_g, _ = strconv.Atoi(os.Getenv("RETRIES"))

	// get capacity
	// incoming.Capacity_g, _ = strconv.ParseFloat(os.Getenv("CAPACITY"), 64)

	// incoming request handling
	// proxy := incoming.NewProxy(globals.RedirectUrl_g)
	inSrv := inServer.NewInServer(globals.RedirectUrl_g)
	inMux := mux.NewRouter()
	inMux.PathPrefix("/").HandlerFunc(inSrv.Handle)

	// outgoing request handling
	outMux := mux.NewRouter()
	outMux.PathPrefix("/").HandlerFunc(outgoing.HandleOutgoing)

	// start running the communication server
	done := make(chan bool)
	defer close(done)
	go controllercomm.RunComm(done)

	// start the in-server
	go inSrv.Run()

	// start the proxy services
	go func() {
		log.Fatal(http.ListenAndServe(globals.PROXYINPORT, inMux))
	}()
	log.Fatal(http.ListenAndServe(globals.PROXOUTPORT, outMux))
}
