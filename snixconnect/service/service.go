//go:build service

package main

import (
	"snixconnect/internal/service"
)

func main() { service.RunSnixConnectService() }
