/*
 * @Author: ybc
 * @Date: 2020-07-22 15:13:29
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-10 20:55:46
 * @Description: file content
 */

package services

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var sig chan os.Signal
var notifySignals []os.Signal

const (
	SIGUSR1 = syscall.Signal(0x10)
)

func init() {
	if runtime.GOOS == "windows" {
		return
	}
	sig = make(chan os.Signal)
	notifySignals = append(notifySignals, syscall.SIGTERM, syscall.SIGUSR1)
	signal.Notify(sig, notifySignals...)
	go handleSignals()
}

func handleSignals() {
	capturedSig := <-sig
	fmt.Println(fmt.Sprintf("Received SIG. [PID:%d, SIG:%v]", syscall.Getpid(), capturedSig))
	switch capturedSig {
	case syscall.SIGTERM:
		close(Exit)
	case syscall.SIGUSR1:
		Reload()
	}
}
