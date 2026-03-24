package main

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"runtime"
)

func openAdminUI(port int) {
	target := fmt.Sprintf("http://127.0.0.1:%d", port)
	if _, err := url.Parse(target); err != nil {
		log.Printf("skip opening admin page: %v", err)
		return
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("open admin page failed: %v", err)
	}
}
