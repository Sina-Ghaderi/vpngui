//go:build !service && !manager

package main

import (
	"snixconnect/internal/handler"
)

func main() { handler.RunSnixConnectApp() }
