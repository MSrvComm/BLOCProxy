package controllercomm

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/MSrvComm/MiCoProxy/globals"
)

// type Endpoint struct {
// 	Svcname string   `json:"Svcname"`
// 	Ips     []string `json:"Ips"`
// }

// var (
// 	SvcList_g = []string{""} // names of all services
// 	// Endpoints_g should be invalidated based on some policy
// 	Endpoints_g = make(map[string][]string) // all endpoints for all services
// )

func GetEndpoints(svcName string) {
	req, err := http.NewRequest("GET", "http://epwatcher:62000/"+svcName, nil)
	if err != nil {
		log.Println("Error reading request. ", err)
	}

	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error getting response: ", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response: ", err.Error())
		return
	}

	var ep globals.Endpoint
	err = json.Unmarshal(body, &ep)
	if err != nil {
		log.Println("error json unmarshalling: ", err.Error())
		return
	}
	// globals.Endpoints_g.Store(ep.Svcname, ep.Ips)
	globals.Endpoints_g[ep.Svcname] = ep.Ips
}

func GetIps(svc string) ([]string, error) {
	resp, err := http.Get("http://epwatcher:62000/" + svc)
	if err != nil {
		log.Println("error getting response: ", err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response: ", err.Error())
		return nil, err
	}

	var ep globals.Endpoint
	err = json.Unmarshal(body, &ep)
	if err != nil {
		log.Println("error json unmarshalling: ", err.Error())
		return nil, err
	}
	return ep.Ips, nil
}

func getAllEndpoints() {
	if len(globals.SvcList_g) > 0 {
		for _, svc := range globals.SvcList_g {
			GetEndpoints(svc)
		}
	}
}

func RunComm(done chan bool) {
	go func() {
		for {
			select {
			case <-time.Tick(time.Microsecond * 10):
				getAllEndpoints()
			case <-done:
				return
			}
		}
	}()
}
