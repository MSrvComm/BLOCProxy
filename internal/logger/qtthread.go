package logger

import (
	"log"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

// type sysItems struct {
// 	avg   float64
// 	count int64
// }

type qItems struct {
	avg   float64
	count int64
}

// type dItems struct {
// 	avg   float64
// 	count int64
// }

type QT struct {
	sitem uint64
	qitem qItems
	ditem uint64
	schan chan bool
	qchan chan int64
	dchan chan bool
}

func NewQT(schan, dchan chan bool, qchan chan int64) *QT {
	return &QT{schan: schan, qchan: qchan, dchan: dchan}
}

func (q *QT) qUpdate(n int64) {
	q.qitem.count++
	q.qitem.avg = q.qitem.avg + (float64((n))-q.qitem.avg)/float64(q.qitem.count)
}

func (q *QT) sUpdate() {
	// q.sitem.count++
	// q.sitem.avg = q.sitem.avg + (float64((q.sitem.count+1))-q.sitem.avg)/float64(q.sitem.count)
	q.sitem++
}

func (q *QT) dUpdate() {
	// q.ditem.count++
	// q.ditem.avg = q.ditem.avg + (float64((q.ditem.count+1))-q.ditem.avg)/float64(q.ditem.count)
	q.ditem++
}

func (q *QT) reset() {
	// use the global capacity if defined
	if globals.CapacityDefined || globals.HardCapaValReached {
		return
	}

	var w float64
	// if q.sitem.avg != 0 {
	// 	w = q.qitem.avg / q.sitem.avg
	// } else {
	// 	w = 0.0
	// }

	// if there was no queue build up
	// we do not know how long the server was idle
	// thus allow capacity to increase
	if q.sitem != 0 && q.qitem.avg != 0 {
		w = q.qitem.avg / float64(q.sitem)
	} else {
		w = 0.0
	}

	log.Printf("Avg Waiting Time: %f seconds\n", w) // debug
	log.Println("Queue Item Avg:", q.qitem.avg)     // debug

	var s float64
	// if q.ditem.avg != 0 {
	// 	s = 1.0 / q.ditem.avg
	// } else {
	// 	s = 0.0
	// }

	// if there was no queue build up
	// we do not know how long the server was idle
	// thus allow capacity to increase
	if q.ditem != 0 && q.qitem.avg != 0 {
		s = 1.0 / float64(q.ditem)
	} else {
		s = 0.0
	}

	log.Println("Service Time Avg:", s) // debug

	totalTime := s + w
	log.Println("Total Time Avg:", totalTime) // debug

	if totalTime != 0 {
		// at the beginning
		if globals.Capacity_g == 0 {
			// globals.Capacity_g = uint64(q.sitem.avg)
			globals.Capacity_g = uint64(q.sitem)
		}

		// capa := uint64(math.Ceil((globals.SLO / totalTime) * float64(globals.Capacity_g)))
		capa := uint64(globals.SLO/totalTime) * globals.Capacity_g
		if capa == 0 {
			globals.Capacity_g = 1
		} else {
			globals.Capacity_g = capa
		}

		if totalTime > globals.SLO {
			globals.HardCapaValReached = true
			log.Println("Maximum Capacity found") // debug
		}

		log.Printf("Capacity is: %d\n", globals.Capacity_g) // debug
	}

	q.qitem.avg = 0.0
	// q.sitem.avg = 0.0
	q.sitem = 0.0
	q.qitem.count = 0
	// q.ditem.count = 0
	q.ditem = 0.0
}

func (q *QT) Run() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			q.reset()
		case n := <-q.qchan:
			q.qUpdate(n)
		case <-q.schan:
			q.sUpdate()
		case <-q.dchan:
			q.dUpdate()
		}
	}
}
