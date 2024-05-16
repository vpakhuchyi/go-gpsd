package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vpakhuchyi/go-gpsd"
)

func main() {
	gps, err := gpsd.Dial(gpsd.DefaultAddress)
	if err != nil {
		log.Fatalf("Failed to connect to GPSD: %s", err)
	}

	gps.Subscribe("GPGGA", func(r interface{}) {
		v := r.(string)
		log.Printf("GPGGA sentence: %s", v)
	})

	gps.Subscribe("GPGSA", func(r interface{}) {
		v := r.(string)
		log.Printf("GPGSA sentence: %s", v)
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	gps.Run("nmea")
	<-sig

	gps.Close()
}
