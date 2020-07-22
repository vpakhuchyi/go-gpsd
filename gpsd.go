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
)

// Filter is a gpsd entry filter function
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
	_, _ = fmt.Fprintf(s.socket, "?WATCH={\"enable\":false}")
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
		_, _ = fmt.Fprintf(s.socket, "?WATCH={\"enable\":true,\"json\":true}")
		s.watch()
		time.Sleep(time.Second)
		_ = s.dial()
	}
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

func (s *Session) watch() {
	// We're not using a JSON decoder because we first need to inspect
	// the JSON string to determine its "class"
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
