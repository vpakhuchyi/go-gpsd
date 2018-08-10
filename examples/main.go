package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	gpsd "github.com/atotto/go-gpsd"
)

func main() {
	gps, err := gpsd.Dial(gpsd.DefaultAddress)
	if err != nil {
		log.Fatalf("Failed to connect to GPSD: %s", err)
	}

	gps.Subscribe("SKY", func(r interface{}) {
		sky := r.(*gpsd.SKYReport)
		log.Printf("%s satellites", len(sky.Satellites))
	})
	gps.Subscribe("TPV", func(r interface{}) {
		tpv := r.(*gpsd.TPVReport)
		log.Printf("mode=%v time=%s", tpv.Mode, tpv.Time)
		log.Printf("Location (%f,%f)", tpv.Lat, tpv.Lon)
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	gps.Run()
	<-sig

	gps.Close()
}
