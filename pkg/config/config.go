package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/MSrvComm/MiCoProxy/pkg/backends"
)

func getPort(e string) int {
	p := os.Getenv(e)
	if p == "" {
		throwFatalErr("No " + e + " port defined")
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		throwFatalErr("Not a valid port:" + e)
	}
	return port
}

func throwFatalErr(msg string) {
	log.Fatal(msg)
}

type Config struct {
	RW         *sync.RWMutex
	ClientPort int
	Inport     int
	Outport    int
	LBPolicy   string
	Services   []string
	BackendMap map[string][]*backends.Backend
}

func NewConfig() *Config {
	return &Config{
		RW:         &sync.RWMutex{},
		ClientPort: getPort("CLIENTPORT"),
		Inport:     getPort("INPORT"),
		Outport:    getPort("OUTPORT"),
		LBPolicy:   os.Getenv("LBPolicy"),
		Services:   make([]string, 0),
		BackendMap: map[string][]*backends.Backend{},
	}
}

func (c *Config) SvcExists(svc string) bool {
	c.RW.RLock()
	defer c.RW.RUnlock()
	_, ok := c.BackendMap[svc]
	return ok
}

func (c *Config) AddNewSvc(svc string, ips []string) {
	if c.SvcExists(svc) {
		return
	}

	c.Services = append(c.Services, svc)
	var backendMap []*backends.Backend
	for i := range ips {
		backendMap = append(backendMap, backends.NewBackend(ips[i]))
	}

	c.RW.Lock()
	defer c.RW.Unlock()
	c.BackendMap[svc] = backendMap
}

func (c *Config) ContainsSrv(ip string) (*backends.Backend, error) {
	// for _, mp := range c.BackendMap {
	// 	for i := range *mp {
	// 		if (*mp)[i].Ip == ip {
	// 			return (*mp)[i], nil
	// 		}
	// 	}
	for svc := range c.BackendMap {
		for i := range c.BackendMap[svc] {
			if c.BackendMap[svc][i].Ip == ip {
				return c.BackendMap[svc][i], nil
			}
		}
	}
	return nil, errors.New("no server found")
}

func (c *Config) AddSrv(svc, ip string) error {
	if !c.SvcExists(svc) {
		return errors.New("no such service")
	}

	c.RW.RLock()
	defer c.RW.RUnlock()
	found := false
	// backendsMap := *c.BackendMap[svc]
	for i := range c.BackendMap[svc] {
		if c.BackendMap[svc][i].Ip == ip {
			found = true
		}
	}
	// for i := range c.BackendMap[svc] {
	// 	if c.BackendMap[svc][i].Ip == ip {
	// 		found = true
	// 	}
	// }

	if !found {
		backend := backends.NewBackend(ip)
		// backendsMap = append(backendsMap, backend)
		c.RW.Lock()
		c.BackendMap[svc] = append(c.BackendMap[svc], backend)
		// c.BackendMap[svc] = &backendsMap
		c.RW.Unlock()
	}

	return nil
}

func (c *Config) backendAlive(svc string, ips []string) {
	c.AddNewSvc(svc, ips)

	c.RW.RLock()
	defer c.RW.RUnlock()
	// backendsMap := *c.BackendMap[svc]
	for s := range c.BackendMap[svc] {
		// for s := range backendsMap {
		// check if the ip is still working
		found := false
		for i := range ips {
			if ips[i] == c.BackendMap[svc][s].Ip {
				// if ips[i] == backendsMap[s].Ip {
				found = true
				break
			}
		}
		// remove the ip
		if !found {
			// c.RW.RUnlock()
			// c.RW.Lock()
			old := c.BackendMap[svc]
			// old := backendsMap
			ln := len(old)
			old[s] = old[ln-1]
			old[ln-1] = nil
			// c.BackendMap[svc] = old[0 : ln-1]
			// backendsMap = old[0 : ln-1]
			c.RW.RUnlock()
			c.RW.Lock()
			// c.BackendMap[svc] = &backendsMap
			c.BackendMap[svc] = old[0 : ln-1]
			c.RW.Unlock()
		}
	}
}

func (c *Config) UpdateMap(svc string, ips []string) {
	c.backendAlive(svc, ips)
	for i := range ips {
		c.AddSrv(svc, ips[i])
	}
}
