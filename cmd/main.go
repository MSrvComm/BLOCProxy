package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/MSrvComm/MiCoProxy/pkg/config"
	"github.com/MSrvComm/MiCoProxy/pkg/controllercomm"
	"github.com/MSrvComm/MiCoProxy/pkg/incoming"
	"github.com/MSrvComm/MiCoProxy/pkg/outgoing"

	"github.com/gin-gonic/gin"
)

func main() {
	conf := config.NewConfig()
	redirecrUrl := "http://localhost:" + strconv.Itoa(conf.ClientPort)
	log.Println("ClientPort:", conf.ClientPort)

	inProxy := *incoming.NewInProxy(redirecrUrl)
	inRouter := gin.Default()
	inRouter.NoRoute(inProxy.Handle)
	inRouter.GET("/", inProxy.Handle)

	outProxy := *outgoing.NewOutProxy(conf)
	outRouter := gin.Default()
	outRouter.GET("/", outProxy.Handle)
	outRouter.NoRoute(outProxy.Handle)

	done := make(chan bool)
	go controllercomm.RunComm(conf, done)

	// go log.Fatal(inRouter.Run("localhost:" + strconv.Itoa(conf.Inport)))

	// log.Fatal(outRouter.Run("localhost:" + strconv.Itoa(conf.Outport)))

	go func() {
		iport := fmt.Sprintf(":%d", conf.Inport)
		log.Println("Proxy Input Port:", iport)
		log.Fatal(inRouter.Run(iport))
	}()

	oport := fmt.Sprintf(":%d", conf.Outport)
	log.Println("Proxy Output Port:", oport)
	log.Fatal(outRouter.Run(oport))
}
