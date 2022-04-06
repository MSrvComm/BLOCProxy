package aqm

import (
	"os"
	"strconv"
	"time"
)

type AQM struct {
	slo time.Duration
}

func NewAQM() *AQM {
	s, _ := strconv.Atoi(os.Getenv("SLO"))
	sl := time.Duration(s) * time.Millisecond
	return &AQM{slo: sl}
}

func (a *AQM) IsOverThreshold(rtt time.Duration) bool {
	return float64(rtt) > 0.8*float64(a.slo)
}
