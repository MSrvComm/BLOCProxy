package outgoing

import (
	"log"
	"net/http"

	"github.com/MSrvComm/MiCoProxy/internal/loadbalancer"
)

type RequestHandler struct {
	send chan *loadbalancer.Request
}

func NewRequestHandler(send chan *loadbalancer.Request) *RequestHandler {
	return &RequestHandler{send}
}

func (rh *RequestHandler) HandleOutgoing(w http.ResponseWriter, r *http.Request) {
	log.Println("HandleOutGoing: Received new request") // debug

	done := make(chan bool)
	defer close(done)
	rqs := loadbalancer.NewRequest(w, r, done)
	rh.send <- rqs
	<-done // block here
}
