/*
 * @Author: ybc
 * @Date: 2020-07-22 15:13:29
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-06 20:11:46
 * @Description: file content
 */

package services

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var sig chan os.Signal
var notifySignals []os.Signal

const (
	SIGUSR1 = syscall.Signal(0x10)
)

func init() {
	sig = make(chan os.Signal)
	notifySignals = append(notifySignals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, SIGUSR1)
	signal.Notify(sig, notifySignals...)
}

func handleSignals() {
	capturedSig := <-sig
	fmt.Println(fmt.Sprintf("Received SIG. [PID:%d, SIG:%v]", syscall.Getpid(), capturedSig))
	switch capturedSig {
	case syscall.SIGHUP:
	case syscall.SIGINT:
		fallthrough
	case syscall.SIGTERM:
		close(Exit)
	case syscall.SIGQUIT:
	case SIGUSR1:
		Reload()
	}
}
