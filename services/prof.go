/*
 * @Author: ybc
 * @Date: 2020-08-14 10:47:41
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-14 10:48:13
 * @Description: file content
 */
package services

import (
	"net/http"
	"net/http/pprof"
)

const (
	pprofAddr string = ":7890"
)

func StartHTTPDebuger() {
	pprofHandler := http.NewServeMux()
	pprofHandler.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	server := &http.Server{Addr: pprofAddr, Handler: pprofHandler}
	go server.ListenAndServe()
}
