/*
 * @Author: ybc
 * @Date: 2020-06-29 19:26:05
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-05 19:43:17
 * @Description: file content
 */

package main

import (
	"file-guard/services"
	"time"
)

func main() {
	go func() {
		time.Sleep(time.Second * 10)
		services.Reload()
	}()

	services.Listen()
}
