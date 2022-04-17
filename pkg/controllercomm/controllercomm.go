package controllercomm

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/MSrvComm/MiCoProxy/pkg/config"
)

type endpoint struct {
	Svcname string   `json:"Svcname"`
	Ips     []string `json:"Ips"`
}

func GetEndpoints(svc string) *[]string {
	// log.Println("Getting endpoints for", svc)
	req, err := http.NewRequest("GET", "http://epwatcher:62000/"+svc, nil)
	if err != nil {
		log.Println("Error reading request:", err)
	}

	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{Timeout: time.Second * 10}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("error getting response:", err.Error())
		return nil
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("error reading response:", err.Error())
		return nil
	}
	var ep endpoint
	err = json.Unmarshal(body, &ep)
	if err != nil {
		log.Println("error json unmarshalling: ", err.Error())
		return nil
	}
	return &ep.Ips
}

func getAllEndpoints(conf *config.Config) {
	// log.Println("Getting all endpoints")
	for _, svc := range conf.Services {
		ips := GetEndpoints(svc)
		conf.UpdateMap(svc, *ips)
	}
}

func RunComm(conf *config.Config, done chan bool) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			getAllEndpoints(conf)
		case <-done:
			return
		}
	}
}
