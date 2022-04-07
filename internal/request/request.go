package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type Request struct {
	d chan bool
	r *http.Request
	w http.ResponseWriter
}

func NewRequest(w http.ResponseWriter, r *http.Request, d chan bool) *Request {
	return &Request{w: w, r: r, d: d}
}

func (rq *Request) MakeRequest(link string) {
	resp, err := http.Get(link)
	if err != nil {
		rq.w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(rq.w, err.Error())
		return
	}
	log.Println(resp)
	for key, values := range resp.Header {
		for _, value := range values {
			rq.w.Header().Set(key, value)
		}
	}
	rq.w.WriteHeader(resp.StatusCode)
	io.Copy(rq.w, resp.Body)
	rq.d <- true
}
