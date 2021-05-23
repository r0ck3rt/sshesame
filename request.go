package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

type tcpipRequestPayload struct {
	Address string
	Port    uint32
}

func (payload tcpipRequestPayload) String() string {
	return net.JoinHostPort(payload.Address, fmt.Sprint(payload.Port))
}

func handleGlobalRequests(requests <-chan *ssh.Request, conn ssh.ConnMetadata) {
	for request := range requests {
		var requestPayload interface{}
		accept := true
		switch request.Type {
		case "tcpip-forward":
			fallthrough
		case "cancel-tcpip-forward":
			requestPayload = new(tcpipRequestPayload)
		default:
			log.Println("Unsupported global request type", request.Type)
			accept = false
		}
		requestPayloadString := ""
		if requestPayload != nil {
			if err := ssh.Unmarshal(request.Payload, requestPayload); err != nil {
				log.Println("Failed to parse request payload", err)
				continue
			}

			requestPayloadString = fmt.Sprint(requestPayload)
		}

		getLogEntry(conn).WithFields(logrus.Fields{
			"request_payload":    requestPayloadString,
			"request_type":       request.Type,
			"request_want_reply": request.WantReply,
			"accepted":           accept,
		}).Infoln("Global request received")

		if request.WantReply {
			var response []byte
			if request.Type == "tcpip-forward" && requestPayload.(*tcpipRequestPayload).Port == 0 {
				response = ssh.Marshal(struct{ port uint32 }{uint32(rand.Intn(65536-1024) + 1024)})
			}
			if err := request.Reply(accept, response); err != nil {
				log.Println("Failed to reply to global request:", err)
				continue
			}
		}
	}
}

type ptyRequestPayload struct {
	Term                                   string
	Width, Height, PixelWidth, PixelHeight uint32
	Modes                                  string
}

func (payload ptyRequestPayload) String() string {
	terminalModes := []string{}
	modeBytes := []byte(payload.Modes)
	for i := 0; i < len(modeBytes); i += 5 {
		opcode := modeBytes[i]
		if opcode >= 160 {
			break
		}
		argument := binary.BigEndian.Uint32(modeBytes[i+1 : i+5])
		terminalModes = append(terminalModes, fmt.Sprintf("%v: %v", opcode, argument))
	}
	return fmt.Sprintf("Term: %v, Size: %vx%v (%vx%v px), Modes: %v", payload.Term, payload.Width, payload.Height, payload.PixelWidth, payload.PixelHeight, strings.Join(terminalModes, ", "))
}

type x11RequestPayload struct {
	SingleConnection         bool
	AuthProtocol, AuthCookie string
	ScreenNumber             uint32
}

func (payload x11RequestPayload) String() string {
	return fmt.Sprintf("Single connection: %v, Auth protocol: %v, Auth cookie: %v, Screen number: %v", payload.SingleConnection, payload.AuthProtocol, payload.AuthCookie, payload.ScreenNumber)
}

type envRequestPayload struct {
	Name, Value string
}

func (payload envRequestPayload) String() string {
	return fmt.Sprintf("%v=%v", payload.Name, payload.Value)
}

type execRequestPayload struct {
	Command string
}

func (payload execRequestPayload) String() string {
	return payload.Command
}

type subsystemRequestPayload struct {
	Subsystem string
}

func (payload subsystemRequestPayload) String() string {
	return payload.Subsystem
}

type windowChangeRequestPayload struct {
	Width, Height, PixelWidth, PixelHeight uint32
}

func (payload windowChangeRequestPayload) String() string {
	return fmt.Sprintf("%vx%v (%vx%v px)", payload.Width, payload.Height, payload.PixelWidth, payload.PixelHeight)
}

type flowControlRequestPayload struct {
	Enabled bool
}

func (payload flowControlRequestPayload) String() string {
	return fmt.Sprintf("Enabled: %v", payload.Enabled)
}

type signalRequestPayload struct {
	Signal string
}

func (payload signalRequestPayload) String() string {
	return payload.Signal
}

type exitStatusRequestPayload struct {
	ExitStatus uint32
}

func (payload exitStatusRequestPayload) String() string {
	return fmt.Sprint(payload.ExitStatus)
}

type exitSignalRequestPayload struct {
	Signal     string
	CoreDumped bool
	Error      string
	Language   string
}

func (payload exitSignalRequestPayload) String() string {
	return fmt.Sprintf("Signal: %v, Core dumped: %v, Error: %v, Language: %v", payload.Signal, payload.CoreDumped, payload.Error, payload.Language)
}

func handleChannelRequests(requests <-chan *ssh.Request, conn channelMetadata) {
	for request := range requests {
		var requestPayload interface{}
		accept := true
		switch request.Type {
		case "pty-req":
			requestPayload = new(ptyRequestPayload)
		case "x11-req":
			requestPayload = new(x11RequestPayload)
		case "env":
			requestPayload = new(envRequestPayload)
		case "shell":
		case "exec":
			requestPayload = new(execRequestPayload)
		case "subsystem":
			requestPayload = new(subsystemRequestPayload)
		case "window-change":
			requestPayload = new(windowChangeRequestPayload)
		case "xon-xoff":
			requestPayload = new(flowControlRequestPayload)
		case "signal":
			requestPayload = new(signalRequestPayload)
		case "exit-status":
			requestPayload = new(exitStatusRequestPayload)
		case "exit-signal":
			requestPayload = new(exitSignalRequestPayload)
		default:
			log.Println("Unsupported channel request type", request.Type)
			accept = false
		}
		requestPayloadString := ""
		if requestPayload != nil {
			if err := ssh.Unmarshal(request.Payload, requestPayload); err != nil {
				log.Println("Failed to parse request payload", err)
				continue
			}

			requestPayloadString = fmt.Sprint(requestPayload)
		}

		conn.getLogEntry().WithFields(logrus.Fields{
			"request_payload":    requestPayloadString,
			"request_type":       request.Type,
			"request_want_reply": request.WantReply,
			"accepted":           accept,
		}).Infoln("Channel request received")

		if request.WantReply {
			if err := request.Reply(accept, nil); err != nil {
				log.Println("Failed to reply to channel request:", err)
				continue
			}
		}
	}
}
