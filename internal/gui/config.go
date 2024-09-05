package gui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unsafe"

	"snixconnect/pkg/walk"

	"golang.org/x/sys/windows"
)

const (
	snixConnectAppDir  = "\\SnixConnect\\"
	appConfigFileName  = "app-config.json"
	credentialFileName = "credentials.json"
	tunDeviceGuid      = "tunnel-guid.bin"
	crashReportFile    = "crash-report.txt"
	filePerm           = 0600
)

type userCredential struct {
	ServerAddress string
	UserCredential
	LastConnected bool
}

type UserCredential struct {
	Username string
	Password string
	Group    string
}

type UserAppConfig struct {
	SkipTLSVerify   bool
	CredentialCache bool
}

const guidStructLen = int(unsafe.Sizeof(windows.GUID{}))

var localAppDirByCmd string

func castGuidToSlice(g *windows.GUID) []byte {
	b := make([]byte, guidStructLen)
	copy(b, unsafe.Slice((*byte)(unsafe.Pointer(g)), guidStructLen))
	return b
}

func castSliceToGuid(b []byte) *windows.GUID {
	if len(b) != guidStructLen {
		return nil
	}
	guid := &windows.GUID{}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(guid)), guidStructLen), b)
	return guid
}

func getTunGuidValue() (guid *windows.GUID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: loading tunnel guid: %v", err)
		}
	}()

	path, err := mkdirLocalAppConfig(localAppDirByCmd)
	if err != nil {
		return nil, err
	}
	flag := os.O_RDWR | os.O_CREATE
	f, err := os.OpenFile(path+tunDeviceGuid, flag, filePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	guidBuff := new(bytes.Buffer)
	_, err = guidBuff.ReadFrom(io.LimitReader(f, int64(guidStructLen)))
	if err != nil {
		return
	}

	if guidBuff.Len() == 0 {
		gNewID, errguid := windows.GenerateGUID()
		if errguid != nil {
			return nil, errguid
		}
		if err := f.Truncate(0); err != nil {
			return nil, err
		}
		if _, err := f.Seek(0, 0); err != nil {
			return nil, err
		}
		if _, err := f.Write(castGuidToSlice(&gNewID)); err != nil {
			return nil, err
		}
		if err := f.Sync(); err != nil {
			return nil, err
		}
		return &gNewID, nil
	}

	if guidBuff.Len() != guidStructLen {
		return nil, fmt.Errorf("invalid win32 guid value has been read from file")
	}

	return castSliceToGuid(guidBuff.Bytes()), nil
}

func CrashReportFile(baseDir string) (file *os.File, path string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: creating crash report file: %v", err)
		}
	}()

	path, err = mkdirLocalAppConfig(baseDir)
	if err != nil {
		return
	}

	path = path + crashReportFile

	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	file, err = os.OpenFile(path, flag, filePerm)
	return
}

func loadUserCerdential() (u *userCredential, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: loading credentials: %v", err)
		}
	}()

	path, err := mkdirLocalAppConfig(localAppDirByCmd)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path+credentialFileName, os.O_RDONLY, filePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var user = new(userCredential)
	return user, json.NewDecoder(f).Decode(user)
}

func loadUserAppConfig() (u *UserAppConfig, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: loading config file: %v", err)
		}
	}()

	path, err := mkdirLocalAppConfig(localAppDirByCmd)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path+appConfigFileName, os.O_RDONLY, filePerm)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config = new(UserAppConfig)
	err = json.NewDecoder(f).Decode(config)
	return config, err
}

func saveUserAppConfig(appconf *UserAppConfig) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: saving config file: %v", err)
		}
	}()

	path, err := mkdirLocalAppConfig(localAppDirByCmd)
	if err != nil {
		return err
	}

	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	f, err := os.OpenFile(path+appConfigFileName, flag, filePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(appconf)
	if err != nil {
		return
	}
	return f.Sync()
}

func mkdirLocalAppConfig(dirPath string) (string, error) {

	if len(dirPath) == 0 {
		dir, err := walk.LocalAppDataPath()
		if err != nil {
			return dir, err
		}
		dirPath = dir
	}
	fullRirPath := dirPath + snixConnectAppDir
	err := os.MkdirAll(fullRirPath, filePerm)

	return fullRirPath, err
}

func saveUserCredential(user *userCredential) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error: saving credentials: %v", err)
		}
	}()
	path, err := mkdirLocalAppConfig(localAppDirByCmd)
	if err != nil {
		return err
	}

	flag := os.O_RDWR | os.O_CREATE | os.O_TRUNC
	f, err := os.OpenFile(path+credentialFileName, flag, filePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(user)
	if err != nil {
		return
	}

	return f.Sync()
}

func removeUserCerdential() error {
	empty := new(userCredential)
	cre, err := loadUserCerdential()
	if err != nil {
		return saveUserCredential(empty)
	}

	empty.ServerAddress = cre.ServerAddress
	return saveUserCredential(empty)
}
