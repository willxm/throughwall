package util

import (
	"fmt"
	"log"
	"os"
	"syscall"
)

func SigHandler(c chan os.Signal) {
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			exitHandler()
		default:
			log.Println("signal ", s)
		}
	}
}

func exitHandler() {
	fmt.Println("exiting graceful ...")
	os.Exit(0)
}
