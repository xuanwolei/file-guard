/*
 * @Author: ybc
 * @Date: 2020-07-22 15:13:29
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-22 15:15:48
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

func init() {
	sig = make(chan os.Signal)
	notifySignals = append(notifySignals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	signal.Notify(sig, notifySignals...)
}

// 捕获系统信号
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
	}
}
