package service

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"snixconnect/pkg/npipe"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

// windows service: dealing with nasty fucking shits:
const snixConnectServiceName = "SnixConnect"
const serviceDescription = "SnixConnect VPN Client Service"
const serviceDescLogn = "SnixConnect Secure And Fast VPN Client For Windows"

const snixConnectPipeName = `\\.\pipe\SnixconnectPipe`
const connExecOK = "OKOK"

var serviceLog debug.Log

func RunSnixConnectService() {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if running in an interactive session: %v", err)
	}

	if !isService {
		log.Fatal("fatal: this is a service, should run with windows service manager")
	}

	runSnixConnectService(snixConnectServiceName, false)
}

func runSnixConnectService(name string, isDebug bool) {
	var err error
	if isDebug {
		serviceLog = debug.New(name)
	} else {
		serviceLog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer serviceLog.Close()
	serviceLog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}

	err = run(name, new(executeSnixAppUnderAndmin))
	if err != nil {
		serviceLog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	serviceLog.Info(1, fmt.Sprintf("%s service stopped", name))
}

type executeSnixAppUnderAndmin struct{}

func (*executeSnixAppUnderAndmin) Execute(args []string, rcvRequest <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	if len(os.Args) < 2 {
		serviceLog.Error(1, "snixconnect app binary path is not specified")
		goto exitService
	}

	go func() {
		err := seerviceCmdLoop()
		if err != nil {
			serviceLog.Error(1, fmt.Sprintf("listen to notify cmd: %v", err))
			log.Fatal(err)
		}
	}()

service:
	for c := range rcvRequest {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
			time.Sleep(100 * time.Millisecond)
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break service
		default:
			serviceLog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
		}
	}

exitService:
	changes <- svc.Status{State: svc.StopPending}
	return
}

func seerviceCmdLoop() error {

	snixconnectPath := os.Args[1]
	exPath := filepath.Dir(snixconnectPath)

	ln, err := npipe.Listen(snixConnectPipeName)
	if err != nil {
		return fmt.Errorf("pipe: %v", err)
	}

	serviceLog.Info(1, "wating for cmd execute notify...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			serviceLog.Info(1, fmt.Sprintf("pipe %v", err))
			continue
		}

		go handleExeNotify(conn, snixconnectPath, exPath)
	}

}

func handleExeNotify(conn net.Conn, excpath, dir string) {
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second))
	defer conn.SetReadDeadline(time.Time{})

	serviceLog.Info(1, fmt.Sprintf("ipc client connected to service: %v", conn.RemoteAddr()))
	buff := make([]byte, 4)

	_, err := conn.Read(buff)
	if err != nil {
		serviceLog.Error(1, fmt.Sprintf("ipc read cmd notify: %v", err))
		return
	}

	sessionID := binary.BigEndian.Uint32(buff)

	serviceLog.Info(1, fmt.Sprintf("running snixconnect %s workdir %s", excpath, dir))

	exec := connExecOK
	err = runBinary(excpath, dir, sessionID)
	if err != nil {
		serviceLog.Error(1, fmt.Sprintf("running snixconnect binary: %v", err))
		exec = "\x21\x21\x21\x21"
	} else {
		serviceLog.Info(1, fmt.Sprintf("snixconnect executed successfully at %v", time.Now()))
	}

	conn.SetWriteDeadline(time.Now().Add(time.Second))
	defer conn.SetWriteDeadline(time.Time{})

	_, err = conn.Write([]byte(exec))
	if err != nil {
		serviceLog.Error(1, fmt.Sprintf("ipc write cmd notify: %v", err))
		return
	}

}
