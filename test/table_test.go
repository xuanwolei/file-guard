/*
 * @Author: ybc
 * @Date: 2020-07-24 16:20:53
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-11 16:50:35
 * @Description: file content
 */

package word

import (
	"file-guard/services"
	"fmt"
	"sync"
	"testing"
)

func TestTableIncrby(t *testing.T) {
	var n sync.WaitGroup
	c := make(chan int)
	table := services.NewXwTable()
	for i := 0; i < 10000; i++ {
		n.Add(1)
		go func(i int) {
			n.Done()
			v := table.Incrby("haha", 1)
			fmt.Println(v)
			if i == 0 {
				table.Expire("haha", 1)
			}
		}(i)
	}

	go func() {
		n.Wait()
		close(c)
	}()
	<-c

	fmt.Println(table.GetInt("haha"))
}

func TestIp(t *testing.T) {
	ip, err := services.GetLocalIp()
	if err != nil {
		t.Error("getLocalIp" + err.Error())
	}
	fmt.Println("ip:", ip)
}
