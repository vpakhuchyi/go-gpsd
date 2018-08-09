package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	gpsd "github.com/atotto/go-gpsd"
)

func main() {
	var gps *gpsd.Session
	var err error

	if gps, err = gpsd.Dial(gpsd.DefaultAddress); err != nil {
		panic(fmt.Sprintf("Failed to connect to GPSD: %s", err))
	}

	gps.Subscribe("TPV", func(r interface{}) {
		tpv := r.(*gpsd.TPVReport)
		fmt.Println("TPV", tpv.Mode, tpv.Time)
	})

	skyfilter := func(r interface{}) {
		sky := r.(*gpsd.SKYReport)

		fmt.Println("SKY", len(sky.Satellites), "satellites")
	}
	tpvFilter := func(r interface{}) {
		report := r.(*gpsd.TPVReport)
		fmt.Println("Location updated", report.Lat, report.Lon)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	gps.Subscribe("SKY", skyfilter)
	gps.Subscribe("TPV", tpvFilter)

	gps.Run()
	<-sig

	gps.Close()
}
