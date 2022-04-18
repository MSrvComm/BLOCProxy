package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/pkg/backends"
	"github.com/MSrvComm/MiCoProxy/pkg/config"
	"github.com/MSrvComm/MiCoProxy/pkg/controllercomm"
	"github.com/MSrvComm/MiCoProxy/pkg/credits"
	"github.com/MSrvComm/MiCoProxy/pkg/incoming"
	"github.com/MSrvComm/MiCoProxy/pkg/outgoing"

	"github.com/gin-gonic/gin"
)

func main() {
	conf := config.NewConfig()
	redirecrUrl := "http://localhost:" + strconv.Itoa(conf.ClientPort)
	log.Println("ClientPort:", conf.ClientPort)

	capacity, _ := strconv.ParseFloat(os.Getenv("CAPACITY"), 64)
	backends.InitCredits = capacity

	creditProxy := credits.NewCreditProxy(conf)

	inProxy := *incoming.NewInProxy(redirecrUrl, creditProxy)
	inRouter := gin.Default()
	inRouter.NoRoute(inProxy.Handle)
	inRouter.POST("/credits", creditProxy.Handle)
	inRouter.GET("/", inProxy.Handle)

	outProxy := *outgoing.NewOutProxy(conf)
	outRouter := gin.Default()
	outRouter.GET("/", outProxy.Handle)
	outRouter.NoRoute(outProxy.Handle)

	done := make(chan bool)
	go controllercomm.RunComm(conf, done)

	go creditProxy.Run(done)

	go func() {
		iport := fmt.Sprintf(":%d", conf.Inport)
		log.Println("Proxy Input Port:", iport)
		log.Fatal(inRouter.Run(iport))
	}()

	oport := fmt.Sprintf(":%d", conf.Outport)
	log.Println("Proxy Output Port:", oport)
	log.Fatal(outRouter.Run(oport))
}
