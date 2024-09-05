package service

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func runBinary(appPath, workDir string, sessionID uint32) error {
	var hToken, dupToken windows.Token

	err := windows.OpenProcessToken(
		windows.CurrentProcess(),
		windows.TOKEN_ALL_ACCESS, &hToken)
	if err != nil {
		return fmt.Errorf("OpenProcessToken: %v", err)
	}

	defer hToken.Close()

	err = windows.DuplicateTokenEx(
		hToken,
		windows.MAXIMUM_ALLOWED,
		nil,
		windows.SecurityImpersonation,
		windows.TokenPrimary,
		&dupToken,
	)

	defer dupToken.Close()

	if err != nil {
		return fmt.Errorf("DuplicateTokenEx: %v", err)
	}

	sidSize := int(unsafe.Sizeof(sessionID))
	sessionIDByte := make([]byte, sidSize)
	copy(sessionIDByte, unsafe.Slice((*byte)(unsafe.Pointer(&sessionID)), sidSize))

	err = windows.SetTokenInformation(
		dupToken,
		windows.TokenSessionId,
		&sessionIDByte[0], uint32(sidSize),
	)

	if err != nil {
		return fmt.Errorf("SetTokenInformation: %v", err)
	}

	var startupInfo windows.StartupInfo
	var processInfo windows.ProcessInformation
	startupInfo.ShowWindow = windows.SW_SHOW
	startupInfo.Desktop = windows.StringToUTF16Ptr("winsta0\\default")

	pEnv := new(uint16)
	err = windows.CreateEnvironmentBlock(&pEnv, dupToken, false)
	if err != nil {
		return fmt.Errorf("CreateEnvironmentBlock: %v", err)
	}

	defer windows.DestroyEnvironmentBlock(pEnv)

	localPath, err := getUserLocalPath(sessionID)
	if err != nil {
		return err
	}

	// commad := fmt.Sprintf("%s \"%s\"", appPath, localPath)

	err = windows.CreateProcessAsUser(
		dupToken,
		windows.StringToUTF16Ptr(appPath),
		windows.StringToUTF16Ptr(localPath),
		nil, nil, false,
		uint32(windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NEW_CONSOLE),
		pEnv,
		windows.StringToUTF16Ptr(workDir),
		&startupInfo,
		&processInfo,
	)

	if err != nil {
		return fmt.Errorf("CreateProcessAsUser: %v", err)
	}
	return nil
}

func getUserLocalPath(session uint32) (localPath string, err error) {

	var userToken windows.Token
	err = windows.WTSQueryUserToken(session, &userToken)
	if err != nil {
		return localPath, fmt.Errorf("CreateProcessAsUser: %v", err)
	}

	defer userToken.Close()
	localPath, err = userToken.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		return localPath, fmt.Errorf("KnownFolderPath: %v", err)
	}
	return
}
