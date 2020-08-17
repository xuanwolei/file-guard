/*
 * @Author: ybc
 * @Date: 2020-07-23 16:05:43
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-17 14:58:07
 * @Description: 过滤规则
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
