package npipe

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func pipeSecurityAttr() (*windows.SecurityAttributes, error) {
	secDesc, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, err
	}

	err = secDesc.SetDACL(nil, true, false)
	if err != nil {
		return nil, err
	}

	err = secDesc.SetControl(windows.SE_DACL_PROTECTED, windows.SE_DACL_PROTECTED)
	if err != nil {
		return nil, err
	}

	sattr := windows.SecurityAttributes{}
	sattr.Length = uint32(unsafe.Sizeof(sattr))
	sattr.SecurityDescriptor = secDesc
	sattr.InheritHandle = 0
	return &sattr, nil

}
