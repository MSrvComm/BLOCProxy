package backends

import (
	"log"
	"math/rand"
	"time"
)

var InitCredits float64

type Backend struct {
	Ip      string
	Reqs    int32 // active requests to this backend currently
	Credits int32
}

func NewBackend(ip string) *Backend {
	var credits int32
	if InitCredits < 1 {
		rand.Seed(int64(time.Now().UTC().Nanosecond()))
		if rand.Float64() < InitCredits {
			credits = 1
		} else {
			credits = 0
		}
	} else {
		credits = int32(InitCredits)
	}
	log.Println("new backend: credits:", credits) // debug
	return &Backend{Ip: ip, Credits: credits}
}
