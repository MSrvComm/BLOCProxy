package logger

import (
	"fmt"
	"log"
	"time"
)

type Data struct {
	Count   int64
	Elapsed time.Duration
}

type ML struct {
	m0   float64
	m    float64
	a    float64 // learning rate
	data chan Data
}

func NewML(data chan Data) *ML {
	return &ML{
		m0:   0,
		m:    1,
		a:    0.1,
		data: data,
	}
}

func (ml *ML) gradient(c, e int64) (float64, float64) {
	return -2 * (float64(e) - (ml.m*float64(c) + ml.m0)), -2 * float64(c) * (float64(e) - (ml.m*float64(c) + ml.m0))
}

func (ml *ML) update(c, e int64) {
	u0, u := ml.gradient(c, e)
	ml.m0 = ml.m0 - ml.a*u0
	ml.m = ml.m - ml.a*u
	ml.a = ml.a * 0.5 // self adjusting learning rate
}

func (ml *ML) Run() {
	for d := range ml.data {
		ml.update(d.Count, d.Elapsed.Nanoseconds())
		msg := fmt.Sprintf("timing actual: elapsed: %v, count: %d", d.Elapsed, d.Count)
		log.Println(msg) // debug
		msg = fmt.Sprintf("timing predicted: elapsed: %v, m0: %f, m: %f", ml.m0+ml.m*float64(d.Count), ml.m0, ml.m)
		log.Println(msg) // debug
	}
}
