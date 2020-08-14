/*
 * @Author: ybc
 * @Date: 2020-06-29 19:26:05
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-14 10:48:41
 * @Description: file content
 */

package main

import (
	"file-guard/services"
)

func main() {
	services.StartHTTPDebuger()
	services.Listen()
}
