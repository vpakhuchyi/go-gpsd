# go-gpsd

*GPSD client for Go.*

## Installation

<pre><code># go get github.com/atotto/go-gpsd</code></pre>

go-gpsd has no external dependencies.

## Usage

go-gpsd is a streaming client for GPSD's JSON service and as such can be used only in async manner unlike clients for other languages which support both async and sync modes.

see [example/main.go](./examples/main.go)

### Currently supported GPSD report types

* `VERSION` (`gpsd.VERSIONReport`)
* `TPV` (`gpsd.TPVReport`)
* `SKY` (`gpsd.SKYReport`)
* `ATT` (`gpsd.ATTReport`)
* `GST` (`gpsd.GSTReport`)
* `PPS` (`gpsd.PPSReport`)
* `Devices` (`gpsd.DEVICESReport`)
* `DEVICE` (`gpsd.DEVICEReport`)
* `ERROR` (`gpsd.ERRORReport`)

## Documentation

For complete library docs, visit [GoDoc.org](http://godoc.org/github.com/atotto/go-gpsd) or take a look at the `gpsd.go` file in this repository.

GPSD's documentation on their JSON protocol can be found at [http://catb.org/gpsd/gpsd_json.html](http://catb.org/gpsd/gpsd_json.html)

To learn more about the Stratoberry Pi project, visit our website at [stratoberry.foi.hr](http://stratoberry.foi.hr).


## License

