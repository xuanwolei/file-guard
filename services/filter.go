/*
 * @Author: ybc
 * @Date: 2020-07-23 16:05:43
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-06 19:37:51
 * @Description: file content
 */
package services

import (
	"regexp"
)

func MatchString(pattern string, text string) bool {
	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		return false
	}
	return matched
}
