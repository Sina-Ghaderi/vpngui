package service

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"snixconnect/internal/gui"
	"snixconnect/pkg/npipe"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const startServiceTimeot = 2 * time.Second

func ServiceManagerHandler() {

	log.SetFlags(0)

	action := flag.String("action", "execute", "service action <install|execute|uninstall>")
	servicePath := flag.String("path", "", "service path to install via service mgr")
	snixPath := flag.String("snixpath", "", "snixconnect executable path")
	flag.Parse()

	switch *action {
	case "install":
		if len(*servicePath) == 0 || len(*snixPath) == 0 {
			log.Fatal("fatal: empty path or exec flag")
		}
		err := setupService(*servicePath, *snixPath)
		if err != nil {
			log.Fatal(err)
		}

	case "uninstall":
		err := removeService(snixConnectServiceName)
		if err != nil {
			log.Fatal(err)
		}
	case "execute":
		err := sendExecSnixConnect()
		if err != nil {
			gui.WinErrorBox(err)
			os.Exit(1)
		}

	default:
		log.Fatal("fatal: bad parameters")
	}

}

func getUserSessionID() (uint32, error) {
	pid := windows.GetCurrentProcessId()
	sessionID := uint32(0)
	err := windows.ProcessIdToSessionId(pid, &sessionID)
	return sessionID, err

}

func sendExecSnixConnect() error {

	sessionID, err := getUserSessionID()
	if err != nil {
		return fmt.Errorf("error getting session id: %v", err)
	}

	pclient, err := npipe.DialTimeout(snixConnectPipeName, time.Second)
	if err != nil {
		return fmt.Errorf("error connecting to pipe: %v\n\nIs SnixConnect service running?", err)
	}

	defer pclient.Close()

	pclient.SetWriteDeadline(time.Now().Add(time.Second))
	defer pclient.SetWriteDeadline(time.Time{})

	buff := make([]byte, 4)
	binary.BigEndian.PutUint32(buff, sessionID)

	_, err = pclient.Write(buff)
	if err != nil {
		return fmt.Errorf("error executing SnixConnect\n\nWrite to pipe: %v", translateEof(err))
	}

	_, err = pclient.Read(buff)
	if err != nil {
		return fmt.Errorf("error executing SnixConnect\n\nRead from pipe: %v", translateEof(err))
	}

	if !bytes.Equal(buff, []byte(connExecOK)) {
		return fmt.Errorf("error executing SnixConnect\n\nService failed to execute SnixConnect binary")
	}
	return nil
}

func translateEof(err error) error {
	if err == io.EOF {
		return fmt.Errorf("communication pipe has been closed unexpectedly")
	}

	return err
}

func setupService(servicePath, snixPath string) error {
	if err := removeService(snixConnectServiceName); err != nil {
		return err
	}

	err := installService(
		snixConnectServiceName,
		serviceDescription,
		serviceDescLogn,
		servicePath, snixPath,
	)

	if err != nil {
		return err
	}

	return startService(snixConnectServiceName)

}

func installService(name, desc, descLong, servicePath, snixPath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("fatal: service %s already exists", name)
	}
	s, err = m.CreateService(name, servicePath, mgr.Config{StartType: 0x02,
		DisplayName: desc, Description: descLong}, snixPath)
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource failed: %s", err)
	}
	return nil
}

func startService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("cannot connect to manager: %v", err)
	}
	defer m.Disconnect()
	lc, err := m.LockStatus()
	if err != nil {
		return fmt.Errorf("service manager: %v", err)
	}

	if lc.IsLocked {
		return fmt.Errorf("service manager locked, holder: %v", lc.Owner)
	}

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("open service: %v", err)
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("could not start the service: %v", err)
	}

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("could not retrieve service status: %v", err)
	}

	timeout := time.Now().Add(startServiceTimeot)
	for status.State != svc.Running {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=Running")
		}
		time.Sleep(250 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

func removeService(name string) error {
	if err := stopService(name); err != nil {
		return fmt.Errorf("stopService failed: %v", err)
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return nil
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("removeEventLogSource failed: %s", err)
	}
	return nil
}

func stopService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("cannot connect to manager: %v", err)
	}
	defer m.Disconnect()
	lc, err := m.LockStatus()
	if err != nil {
		return fmt.Errorf("service manager: %v", err)
	}

	if lc.IsLocked {
		return fmt.Errorf("service manager locked, holder: %v", lc.Owner)
	}

	s, err := m.OpenService(name)
	if err != nil {
		return nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("could not retrieve service status: %v", err)
	}
	if status.State == svc.Stopped {
		return nil
	}
	status, err = s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("could not send control=%s: %v", "Stop", err)
	}
	timeout := time.Now().Add(startServiceTimeot)
	for status.State != svc.Stopped {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=Stopped")
		}
		time.Sleep(250 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil

}
