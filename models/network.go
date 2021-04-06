package models

import (
	"fmt"
	"net"
	"os"
)

var (
	outboundIP     = ""    //nolint:gochecknoglobals
	haveOutboundIP = false //nolint:gochecknoglobals
)

// GetOutboundIP Get preferred outbound ip of this machine
func GetOutboundIP() string {
	if haveOutboundIP {
		return outboundIP
	}

	if k, b := os.LookupEnv("AIWARE_HOST_IP"); b {
		haveOutboundIP = true
		outboundIP = k
		return k
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		fmt.Printf("Failed to determine IP: %v", err)
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	haveOutboundIP = true
	outboundIP = localAddr.IP.String()

	return outboundIP
}
