# go-gpsd

*GPSD client for Go.*

## Installation

`go get github.com/vpakhuchyi/go-gpsd`

go-gpsd has no external dependencies.

## Usage

go-gpsd is a streaming client for GPSD's JSON/NMEA service and as such can be used only in async manner unlike clients for other languages which support both async and sync modes.

see [example/main.go](./examples/main.go)

### Current supported GPSD report types

* `VERSION` (`gpsd.VERSIONReport`)
* `TPV` (`gpsd.TPVReport`)
* `SKY` (`gpsd.SKYReport`)
* `ATT` (`gpsd.ATTReport`)
* `GST` (`gpsd.GSTReport`)
* `PPS` (`gpsd.PPSReport`)
* `Devices` (`gpsd.DEVICESReport`)
* `DEVICE` (`gpsd.DEVICEReport`)
* `ERROR` (`gpsd.ERRORReport`)

## License

MIT License
https://github.com/vpakhuchyi/go-gpsd/blob/master/LICENSE