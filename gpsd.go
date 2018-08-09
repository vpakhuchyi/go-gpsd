package gpsd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// DefaultAddress of gpsd (localhost:2947)
const DefaultAddress = "localhost:2947"

// Filter is a gpsd entry filter function
type Filter func(interface{})

// Session represents a connection to gpsd
type Session struct {
	address string
	socket  net.Conn
	reader  *bufio.Reader
	filters map[string][]Filter

	done chan struct{}
}

// Mode describes status of a TPV report
type Mode byte

const (
	// NoValueSeen indicates no data has been received yet
	NoValueSeen Mode = 0
	// NoFix indicates fix has not been required yet
	NoFix Mode = 1
	// Mode2D represents quality of the fix
	Mode2D Mode = 2
	// Mode3D represents quality of the fix
	Mode3D Mode = 3
)

type gpsdReport struct {
	Class string `json:"class"`
}

// TPVReport is a Time-Position-Velocity report
type TPVReport struct {
	Class  string    `json:"class"`
	Tag    string    `json:"tag"`
	Device string    `json:"device"`
	Mode   Mode      `json:"mode"`
	Time   time.Time `json:"time"`
	Ept    float64   `json:"ept"`
	Lat    float64   `json:"lat"`
	Lon    float64   `json:"lon"`
	Alt    float64   `json:"alt"`
	Epx    float64   `json:"epx"`
	Epy    float64   `json:"epy"`
	Epv    float64   `json:"epv"`
	Track  float64   `json:"track"`
	Speed  float64   `json:"speed"`
	Climb  float64   `json:"climb"`
	Epd    float64   `json:"epd"`
	Eps    float64   `json:"eps"`
	Epc    float64   `json:"epc"`
}

// SKYReport reports sky view of GPS satellites
type SKYReport struct {
	Class      string      `json:"class"`
	Tag        string      `json:"tag"`
	Device     string      `json:"device"`
	Time       time.Time   `json:"time"`
	Xdop       float64     `json:"xdop"`
	Ydop       float64     `json:"ydop"`
	Vdop       float64     `json:"vdop"`
	Tdop       float64     `json:"tdop"`
	Hdop       float64     `json:"hdop"`
	Pdop       float64     `json:"pdop"`
	Gdop       float64     `json:"gdop"`
	Satellites []Satellite `json:"satellites"`
}

// GSTReport is pseudorange noise report
type GSTReport struct {
	Class  string    `json:"class"`
	Tag    string    `json:"tag"`
	Device string    `json:"device"`
	Time   time.Time `json:"time"`
	Rms    float64   `json:"rms"`
	Major  float64   `json:"major"`
	Minor  float64   `json:"minor"`
	Orient float64   `json:"orient"`
	Lat    float64   `json:"lat"`
	Lon    float64   `json:"lon"`
	Alt    float64   `json:"alt"`
}

// ATTReport reports vehicle-attitude from the digital compass or the gyroscope
type ATTReport struct {
	Class       string    `json:"class"`
	Tag         string    `json:"tag"`
	Device      string    `json:"device"`
	Time        time.Time `json:"time"`
	Heading     float64   `json:"heading"`
	MagSt       string    `json:"mag_st"`
	Pitch       float64   `json:"pitch"`
	PitchSt     string    `json:"pitch_st"`
	Yaw         float64   `json:"yaw"`
	YawSt       string    `json:"yaw_st"`
	Roll        float64   `json:"roll"`
	RollSt      string    `json:"roll_st"`
	Dip         float64   `json:"dip"`
	MagLen      float64   `json:"mag_len"`
	MagX        float64   `json:"mag_x"`
	MagY        float64   `json:"mag_y"`
	MagZ        float64   `json:"mag_z"`
	AccLen      float64   `json:"acc_len"`
	AccX        float64   `json:"acc_x"`
	AccY        float64   `json:"acc_y"`
	AccZ        float64   `json:"acc_z"`
	GyroX       float64   `json:"gyro_x"`
	GyroY       float64   `json:"gyro_y"`
	Depth       float64   `json:"depth"`
	Temperature float64   `json:"temperature"`
}

// VERSIONReport returns version details of gpsd client
type VERSIONReport struct {
	Class      string `json:"class"`
	Release    string `json:"release"`
	Rev        string `json:"rev"`
	ProtoMajor int    `json:"proto_major"`
	ProtoMinor int    `json:"proto_minor"`
	Remote     string `json:"remote"`
}

// DEVICESReport lists all devices connected to the system
type DEVICESReport struct {
	Class   string         `json:"class"`
	Devices []DEVICEReport `json:"devices"`
	Remote  string         `json:"remote"`
}

// DEVICEReport reports a state of a particular device
type DEVICEReport struct {
	Class     string  `json:"class"`
	Path      string  `json:"path"`
	Activated string  `json:"activated"`
	Flags     int     `json:"flags"`
	Driver    string  `json:"driver"`
	Subtype   string  `json:"subtype"`
	Bps       int     `json:"bps"`
	Parity    string  `json:"parity"`
	Stopbits  string  `json:"stopbits"`
	Native    int     `json:"native"`
	Cycle     float64 `json:"cycle"`
	Mincycle  float64 `json:"mincycle"`
}

// PPSReport is triggered on each pulse-per-second strobe from a device
type PPSReport struct {
	Class      string  `json:"class"`
	Device     string  `json:"device"`
	RealSec    float64 `json:"real_sec"`
	RealMusec  float64 `json:"real_musec"`
	ClockSec   float64 `json:"clock_sec"`
	ClockMusec float64 `json:"clock_musec"`
}

// ERRORReport is an error response
type ERRORReport struct {
	Class   string `json:"class"`
	Message string `json:"message"`
}

// Satellite describes a location of a GPS satellite
type Satellite struct {
	PRN  float64 `json:"PRN"`
	Az   float64 `json:"az"`
	El   float64 `json:"el"`
	Ss   float64 `json:"ss"`
	Used bool    `json:"used"`
}

// Dial opens a new connection to GPSD.
func Dial(address string) (*Session, error) {
	s := &Session{
		address: address,
	}
	if err := s.dial(); err != nil {
		return nil, err
	}
	s.filters = make(map[string][]Filter)

	return s, nil
}

func (s *Session) dial() error {
	conn, err := net.Dial("tcp4", s.address)
	if err != nil {
		return err
	}

	s.socket = conn
	s.reader = bufio.NewReader(conn)
	_, err = s.reader.ReadString('\n')
	return err
}

func (s *Session) Close() error {
	close(s.done)
	return s.socket.Close()
}

func (s *Session) Run() {
	go s.run()
}

func (s *Session) run() {
	s.done = make(chan struct{})

	for {
		select {
		case <-s.done:
			return
		default:
		}
		fmt.Fprintf(s.socket, "?WATCH={\"enable\":true,\"json\":true}")
		s.watch()
		time.Sleep(time.Second)
		s.dial()
	}
}

// SendCommand sends a command to GPSD
func (s *Session) SendCommand(command string) {
	fmt.Fprintf(s.socket, "?"+command+";")
}

func (s *Session) Subscribe(class string, f Filter) {
	s.filters[class] = append(s.filters[class], f)
}

func (s *Session) deliverReport(class string, report interface{}) {
	for _, f := range s.filters[class] {
		f(report)
	}
}

func (s *Session) watch() {
	// We're not using a JSON decoder because we first need to inspect
	// the JSON string to determine it's "class"
	for {
		select {
		case <-s.done:
			return
		default:
		}
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			if op, ok := err.(*net.OpError); ok && strings.Contains(op.Err.Error(), "use of closed network connection") {
				return
			}
			fmt.Printf("Stream reader error (is gpsd running?): %#v\n", err)
			return
		}

		var reportPeek gpsdReport
		lineBytes := []byte(line)
		if err := json.Unmarshal(lineBytes, &reportPeek); err != nil {
			fmt.Printf("failed to json unmarshal: %s\n", err)
			continue
		}
		if len(s.filters[reportPeek.Class]) == 0 {
			continue
		}

		report, err := unmarshalReport(reportPeek.Class, lineBytes)
		if err != nil {
			fmt.Printf("failed to unmarshal report: %s\n", err)
			continue
		}
		s.deliverReport(reportPeek.Class, report)
	}
}

func unmarshalReport(class string, bytes []byte) (interface{}, error) {
	var err error

	switch class {
	case "TPV":
		var r *TPVReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "SKY":
		var r *SKYReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "GST":
		var r *GSTReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "ATT":
		var r *ATTReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "VERSION":
		var r *VERSIONReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "DEVICES":
		var r *DEVICESReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "PPS":
		var r *PPSReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	case "ERROR":
		var r *ERRORReport
		err = json.Unmarshal(bytes, &r)
		return r, err
	}

	return nil, err
}
