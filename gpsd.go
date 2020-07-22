/*
Package gpsd is a streaming client for GPSD's JSON service and as such can be used only in
async manner unlike clients for other languages which support both async and sync modes.
*/
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

const (
	// DefaultAddress of gpsd
	DefaultAddress = "localhost:2947"

	// dialTimeout is how long the client will wait for gpsd
	dialTimeout = 2 * time.Second

	WatchCommand   = "WATCH"
	PollCommand    = "POLL"
	VersionCommand = "VERSION"
)

// Filter is a gpsd entry filter function (aka watcher or subscriber)
type Filter func(interface{})

// Session represents a connection to gpsd
type Session struct {
	address string
	socket  net.Conn
	reader  *bufio.Reader
	filters map[string][]Filter
	done    chan struct{}
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
	conn, err := net.DialTimeout("tcp4", s.address, dialTimeout)
	if err != nil {
		return err
	}

	s.socket = conn
	s.reader = bufio.NewReader(conn)
	_, err = s.reader.ReadString('\n')
	return err
}

// Close closes the connection to GPSD
func (s *Session) Close() error {
	s.Watch(map[string]bool{"enable": false})
	close(s.done)
	return s.socket.Close()
}

// Run starts monitoring the connection to GPSD
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
		s.Watch(map[string]bool{"enable": true, "json": true})
		s.watch()
		time.Sleep(time.Second)
		_ = s.dial()
	}
}

// VersionSync sends the version command and returns the version response string
func (s *Session) VersionSync() string {
	s.Version()
	line, _ := s.readLine()
	return line
}

// Version sends the version command to GPSD
func (s *Session) Version() {
	s.SendCommand(VersionCommand)
}

// PollSync sends the poll command returns the poll response string
func (s *Session) PollSync() string {
	s.Poll()
	line, _ := s.readLine()
	return line
}

// Poll sends the poll command to GPSD
func (s *Session) Poll() {
	s.SendCommand(PollCommand)
}

// WatchSync sends the watch command with an optional param object parsed into
// the payload and returns the watch response string
func (s *Session) WatchSync(watchObject ...map[string]bool) string {
	s.Watch(watchObject...)
	line, _ := s.readLine()
	return line
}

// Watch sends the watch command with an optional param object parsed into the payload
func (s *Session) Watch(watchObject ...map[string]bool) {
	objectString := ""
	if len(watchObject) == 1 {
		var values []string
		for k, v := range watchObject[0] {
			values = append(values, fmt.Sprintf(`"%s":%v`, k, v))
		}
		objectString = fmt.Sprintf(`={%s}`, strings.Join(values, ","))
	}
	s.SendCommand(WatchCommand + objectString)
}

// SendCommand sends a command to GPSD
func (s *Session) SendCommand(command string) {
	_, _ = fmt.Fprintf(s.socket, "?"+command+";")
}

func (s *Session) Subscribe(class string, f Filter) {
	s.filters[class] = append(s.filters[class], f)
}

func (s *Session) deliverReport(class string, report interface{}) {
	for _, f := range s.filters[class] {
		f(report)
	}
}

// readLine reads a line from the reader and returns the string
func (s *Session) readLine() (line string, err error) {
	line, err = s.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
		} else if op, ok := err.(*net.OpError); ok && strings.Contains(
			op.Err.Error(), "use of closed network connection") {
		} else {
			fmt.Printf("Stream reader error (is gpsd running?): %#v\n", err)
		}
	}
	return
}

// getClass returns the class string for the passed line in case of error, a blank string is returned
func getClass(line []byte) string {
	var reportPeek gpsdReport
	if err := json.Unmarshal(line, &reportPeek); err != nil {
		fmt.Printf("failed to parse class type: %s\n", err)
		return ""
	}
	return reportPeek.Class
}

func (s *Session) watch() {
	// We're not using a JSON decoder because we first need to inspect
	// the JSON string to determine its "class"
	for {
		select {
		case <-s.done:
			return
		default:
		}
		line, err := s.readLine()
		if err != nil {
			return
		}

		lineBytes := []byte(line)
		class := getClass(lineBytes)

		if len(s.filters[class]) == 0 {
			continue
		}

		report, err := unmarshalReport(class, lineBytes)
		if err != nil {
			fmt.Printf("failed to unmarshal report: %s\n", err)
			continue
		}

		s.deliverReport(class, report)
	}
}

func unmarshalReport(class string, bytes []byte) (r interface{}, err error) {
	switch class {
	case "TPV":
		r = new(TPVReport)
	case "SKY":
		r = new(SKYReport)
	case "GST":
		r = new(GSTReport)
	case "ATT":
		r = new(ATTReport)
	case "VERSION":
		r = new(VERSIONReport)
	case "DEVICES":
		r = new(DEVICESReport)
	case "PPS":
		r = new(PPSReport)
	case "ERROR":
		r = new(ERRORReport)
	}
	return r, json.Unmarshal(bytes, &r)
}
