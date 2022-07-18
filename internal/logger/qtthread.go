package logger

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

type QT struct {
	sitem    uint64
	qitem    qItems
	ditem    uint64
	lastStep float64
	schan    chan bool
	qchan    chan int64
	dchan    chan bool
}

func NewQT(schan, dchan chan bool, qchan chan int64) *QT {
	return &QT{schan: schan, qchan: qchan, dchan: dchan}
}

func (q *QT) qUpdate(n int64) {
	q.qitem.count++
	q.qitem.avg = q.qitem.avg + (float64((n))-q.qitem.avg)/float64(q.qitem.count)
}

func (q *QT) sUpdate() {
	q.sitem++
}

func (q *QT) dUpdate() {
	q.ditem++
}

func (q *QT) update() {
	// use global capacity if defined
	if globals.CapacityDefined {
		return
	}

	// precaution against capacity being set to 0
	if globals.Capacity_g == 0 {
		globals.Capacity_g = 1
	}

	if q.ditem != 0 {
		if 1.0/float64(q.ditem) > globals.SLO {
			if q.sitem == q.ditem { // likely lot of idle time
				log.Println("s->item == d->item", q.ditem) // debug
				globals.Capacity_g = q.ditem
			} else {
				log.Println("s->item != d->item", q.qitem.avg)
				globals.Capacity_g = uint64(math.Ceil(q.qitem.avg))
			}
		}
	}

	// // calculate wait time
	// w := 0.0
	// if q.sitem != 0 {
	// 	w = q.qitem.avg / float64(q.sitem)
	// }
	// log.Printf("Avg Waiting Time: %f seconds\n", w) // debug
	// log.Println("Queue Item Avg:", q.qitem.avg)     // debug

	// // we want the wait time to be less than half of SLO
	// slo := globals.SLO * 0.9
	// if w != 0.0 { // don't update capacity if nothing is happening
	// 	r := (slo - w) / slo
	// 	step := math.Ceil(float64(globals.Capacity_g) * r)
	// 	log.Println("Step:", step) // debug
	// 	deltaStep := step - q.lastStep
	// 	log.Println("Delta Step:", deltaStep) // debug
	// 	q.lastStep = step
	// 	step += deltaStep

	// 	capa := float64(globals.Capacity_g) + step
	// 	if capa < 0 {
	// 		globals.Capacity_g = 1
	// 	} else {
	// 		globals.Capacity_g = uint64(capa)
	// 	}
	// }

	log.Printf("Capacity is: %d\n", globals.Capacity_g) // debug

	q.qitem.avg = 0.0
	q.sitem = 0.0
	q.qitem.count = 0
	q.ditem = 0.0
}

func (q *QT) reset() {
	// use the global capacity if defined
	if globals.CapacityDefined || globals.HardCapaValReached {
		return
	}

	var w float64

	if q.sitem != 0 {
		w = q.qitem.avg / float64(q.sitem)
	} else {
		w = 0.0
	}

	log.Printf("Avg Waiting Time: %f seconds\n", w) // debug
	log.Println("Queue Item Avg:", q.qitem.avg)     // debug

	var s float64
	if q.ditem != 0 && q.sitem != q.ditem {
		s = 1.0 / float64(q.ditem)
	} else {
		s = 0.0
	}

	log.Println("Service Time Avg:", s) // debug

	totalTime := s + w
	log.Println("Total Time Avg:", totalTime) // debug

	// // if totalTime != 0 {
	// if totalTime < globals.SLO && uint64(q.sitem) > globals.Capacity_g {
	// 	q.lastCapa = globals.Capacity_g
	// 	globals.Capacity_g = uint64(q.sitem)
	// } else {
	// 	if globals.Capacity_g != q.lastCapa {
	// 		globals.Capacity_g = q.lastCapa
	// 	} else {
	// 		globals.Capacity_g = uint64(math.Max(float64(globals.Capacity_g/2), 1.0))
	// 	}
	// }

	log.Printf("Capacity is: %d\n", globals.Capacity_g) // debug

	q.qitem.avg = 0.0
	q.sitem = 0.0
	q.qitem.count = 0
	q.ditem = 0.0
}

func (q *QT) Run() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			// q.reset()
			q.update()
		case n := <-q.qchan:
			q.qUpdate(n)
		case <-q.schan:
			q.sUpdate()
		case <-q.dchan:
			q.dUpdate()
		}
	}
}
