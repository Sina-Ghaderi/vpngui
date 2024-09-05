package service

import (
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"
)

func NtGetNextProcess(handle uintptr, MAX_ALLOWED int, param3 int, param4 int, outHandle uintptr) uintptr {
	ret, _, _ := syscall.NewLazyDLL("ntdll.dll").NewProc("NtGetNextProcess").Call(
		uintptr(handle),
		uintptr(MAX_ALLOWED),
		uintptr(param3),
		uintptr(param4),
		uintptr(outHandle),
	)
	return uintptr(ret)
}

func GetProcessImageFileName(hProcess uintptr, lpImageFileName uintptr, nSize int) uintptr {
	ret, _, _ := syscall.NewLazyDLL("psapi.dll").NewProc("GetProcessImageFileNameA").Call(
		uintptr(hProcess),
		uintptr(lpImageFileName),
		uintptr(nSize),
	)
	return uintptr(ret)
}

func GetProcessId(handle uintptr) uintptr {
	ret, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetProcessId").Call(
		uintptr(handle),
	)
	return uintptr(ret)
}

func Reverse(s string) string {
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}

func GetProcessByName(process_name string) []uintptr {
	var proc_handles_slice []uintptr
	var MAXIMUM_ALLOWED int = 0x02000000
	var s uintptr = 0
	for {
		if NtGetNextProcess(s, MAXIMUM_ALLOWED, 0, 0, uintptr(unsafe.Pointer(&s))) != 0 {
			break
		}

		buf := [256]byte{}
		var mem_address uintptr = uintptr(unsafe.Pointer(&buf[0]))
		var res uintptr = GetProcessImageFileName(s, mem_address, len(buf))

		if res > 1 {
			var res_string string = string(buf[0:res])
			var reverted_string string = Reverse(res_string)
			var index int = strings.Index(reverted_string, "\\")
			var result_name string = Reverse(reverted_string[0:index])
			if result_name == process_name {
				// fmt.Println("[+] Process handle: \t", s, "(", fmt.Sprintf("0x%x", s), ")");
				// fmt.Println("[+] Process name:   \t", process_name);
				proc_handles_slice = append(proc_handles_slice, s)
			}
		}
	}
	return proc_handles_slice
}
