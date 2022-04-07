package outgoing

import (
	"log"
	"net/http"

	irequest "github.com/MSrvComm/MiCoProxy/internal/request"
)

type RequestHandler struct {
	send chan *irequest.Request
}

func NewRequestHandler(send chan *irequest.Request) *RequestHandler {
	return &RequestHandler{send}
}

func (rh *RequestHandler) HandleOutgoing(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleOutGoing: Received new request") // debug
	done := make(chan bool)
	defer close(done)
	rqs := irequest.NewRequest(w, r, done)
	rh.send <- rqs
	<-done // block here
}
