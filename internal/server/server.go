package server

import (
	"log"
	"math"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

type qItems struct {
	avg   float64
	count int64
}

type elapsed struct {
	avg   float64
	count int64
}

type Server struct {
	sitem uint64
	qitem qItems
	ditem uint64
	schan chan bool
	qchan chan int64
	dchan chan bool
	echan chan time.Duration
	rtt   elapsed
}

func NewQT(schan, dchan chan bool, qchan chan int64, echan chan time.Duration) *Server {
	return &Server{schan: schan, qchan: qchan, dchan: dchan, echan: echan}
}

func (srv *Server) qUpdate(n int64) {
	tot := srv.qitem.avg*float64(srv.qitem.count) + float64(n)
	srv.qitem.count++
	srv.qitem.avg = tot / float64(srv.qitem.count)
	log.Println("Average queue size:", srv.qitem.avg)
	// srv.qitem.avg = srv.qitem.avg + (float64((n))-srv.qitem.avg)/float64(srv.qitem.count)
}

func (srv *Server) sUpdate() {
	srv.sitem++
}

func (srv *Server) dUpdate() {
	srv.ditem++
}

func (srv *Server) update() {
	// use global capacity if defined
	if globals.CapacityDefined {
		return
	}

	w := 0.0
	if srv.qitem.avg != 0 && srv.sitem != 0 {
		w = srv.qitem.avg / float64(srv.sitem)
	}
	log.Println("Average queue size:", w)

	// precaution against capacity being set to 0
	if globals.Capacity_g == 0 {
		globals.Capacity_g = 1
	}

	/* this works */
	if srv.ditem != 0 {
		// if 1.0/float64(srv.ditem) > globals.SLO {
		// // this number 0.9 controls how much queuing we will allow
		// if 0.9*float64(srv.sitem) < float64(srv.ditem) {
		// globals.Capacity_g = srv.ditem * 2
		// } else {
		if time.Duration(srv.rtt.avg) > time.Duration(globals.SLO*1e+09) {
			globals.Capacity_g = uint64(math.Floor(float64(globals.Capacity_g) / 2.0))
			// }
		} else {
			globals.Capacity_g = srv.ditem * 2
		}
	}

	srv.qitem.avg = 0.0
	srv.sitem = 0.0
	srv.qitem.count = 0
	srv.ditem = 0.0
	srv.rtt.avg = 0.0
	srv.rtt.count = 0
}

func (srv *Server) eUpdate(e time.Duration) {
	tot := srv.rtt.avg + float64(e.Nanoseconds())
	srv.rtt.count++
	srv.rtt.avg = tot / float64(srv.rtt.count)
	// srv.rtt.avg = srv.qitem.avg + (float64((e.Nanoseconds()))-srv.rtt.avg)/float64(srv.rtt.count)
	log.Println("Average elapsed time:", time.Duration(srv.rtt.avg))
}

func (srv *Server) Run() {
	d := time.Second
	ticker := time.NewTicker(d)
	numRequests := 0
	for {
		select {
		case <-ticker.C:
			// // update capacity after 10 requests
			// // adjust ticker to the time taken by 10 requests
			// if numRequests != 0 {
			// 	t := time.Duration(math.Ceil(float64(d) / float64(numRequests))) // time per request
			// 	d = t * 10
			// 	ticker.Reset(d)
			// }
			srv.update()
		case n := <-srv.qchan:
			srv.qUpdate(n)
		case <-srv.schan:
			srv.sUpdate()
		case <-srv.dchan:
			numRequests++
			srv.dUpdate()
		case e := <-srv.echan:
			srv.eUpdate(e)
		}
	}
}
