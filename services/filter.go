/*
 * @Author: ybc
 * @Date: 2020-07-23 16:05:43
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-23 16:28:19
 * @Description: file content
 */
package services

import (
	"regexp"
)

func MatchString(pattern string, text string) bool {
	matched, err := regexp.MatchString("(?i)error", text)
	if err != nil {
		return false
	}
	return matched
}
