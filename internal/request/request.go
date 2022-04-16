package request

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type Request struct {
	Mu        *sync.Mutex
	Rq        *http.Request
	Wr        http.ResponseWriter
	StartTime time.Time
	Priority  time.Duration
	Index     int
	Done      chan bool
}

func NewRequest(w http.ResponseWriter, r *http.Request, done chan bool) *Request {
	var tm time.Time
	stm := r.Context().Value("startTime")
	if stm == nil {
		ntm := time.Now()
		tm = ntm
		ctx := context.WithValue(r.Context(), "startTime", tm)
		r = r.WithContext(ctx)
	} else {
		tm = stm.(time.Time)
	}
	return &Request{Mu: &sync.Mutex{}, Wr: w, Rq: r, Done: done, Priority: time.Since(tm), StartTime: tm}
}

// func (r *Request) SetPriority() {
// 	tm := r.Rq.Context().Value("startTime").(time.Time)
// 	r.Priority = time.Since(tm)
// }
