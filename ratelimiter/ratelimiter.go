/* This rate limiter has been largely influenced by the message publishing rate limiter
 * developed at Ably.
 * https://medium.com/ably-realtime/how-adopting-a-distributed-rate-limiting-helps-scale-your-platform-1afdf3944b5a
 */

package ratelimiter

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

var (
	// totalReqs uint64  // total requests in the last window
	Capacity float64 // selecting a number randomly for now
	clients  clientsStruct
	slope    = 0.33
	Window   int
	// totalReqs uint64
)

type client struct {
	mu          sync.Mutex
	ip          string
	reqSent     float64
	probReject  float64
	lastUpdated time.Time
}

func (c *client) updateRejectRate(ln int, totalReqs float64) {
	log.Println("updateRejectRate: Capacity:", Capacity)
	log.Println("updateRejectRate: Length:", ln)
	log.Println("updateRejectRate: ReqsSent:", c.reqSent)
	c.mu.Lock()
	defer c.mu.Unlock()
	ratio := c.reqSent / Capacity
	// ratio := c.reqSent / totalReqs
	norm := Capacity / float64(ln)
	diff := norm - ratio
	if diff < 0 {
		c.probReject = 0
	} else {
		c.probReject = (diff / norm) * slope
	}
	// c.probReject = (c.reqSent / Capacity / float64(ln)) * slope
	log.Println("RateLimiter reject rate:", c.probReject, "for client:", c.ip)
}

type clientsStruct struct {
	mu      sync.Mutex
	clients []*client
}

func NewClients() {
	clients = clientsStruct{mu: sync.Mutex{}, clients: make([]*client, 0)}
}

func (c *clientsStruct) getLen() int {
	return len(c.clients)
}

func (c *clientsStruct) addClient(ip string) {
	clnt := &client{ip: ip, reqSent: 0, probReject: 0, lastUpdated: time.Now()}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.clients = append(c.clients, clnt)
}

func (c *clientsStruct) search(ip string) *client {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.clients {
		if c.clients[i].ip == ip {
			return c.clients[i]
		}
	}
	return nil
}

func (c *clientsStruct) totalReqs() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := 0.0
	for i := range c.clients {
		s += c.clients[i].reqSent
	}
	return s
}

func (c *clientsStruct) update(ip string) {
	// log.Println("RateLimiter update called with", ip)
	clnt := c.search(ip)
	if clnt == nil {
		return
	}

	clnt.mu.Lock()
	// log.Println("After update lock step")
	ts := time.Since(clnt.lastUpdated)
	log.Println("update: ts:", ts)
	if ts > time.Second {
		clnt.lastUpdated = time.Now()
		clnt.reqSent = 1
	} else {
		clnt.reqSent++
	}
	clnt.mu.Unlock()
	totalReqs := c.totalReqs()
	clnt.updateRejectRate(c.getLen(), totalReqs)
}

func RejectRequest(ip string) bool {
	clients.update(ip)
	// log.Println("RateLimiter returned from update")
	clnt := clients.search(ip)
	if clnt == nil {
		// log.Println("Registering client:", ip)
		clients.addClient(ip)
		return true
	}
	rand.Seed(int64(time.Now().Nanosecond()))
	// log.Println("RjectRequest sending back:", rand.Float64() < clnt.probReject)
	return rand.Float64() < clnt.probReject
}

// func AQM() {
// 	for {
// 		select {
// 		case <-time.Tick(time.Microsecond * 10):

// 		}
// 	}
// }
