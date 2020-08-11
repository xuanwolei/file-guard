/*
 * @Author: ybc
 * @Date: 2020-07-22 15:51:25
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-11 16:52:12
 * @Description: file content
 */

package services

import (
	"io/ioutil"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

type FileInfo struct {
	File os.FileInfo
	Path string
}

func PathExists(path string) (os.FileInfo, error) {
	file, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if os.IsNotExist(err) {
		return nil, err
	}

	return file, nil
}

func FindFiles(path string, file chan<- *FileInfo, isMatch bool) {
	dir, name := ParseFilePath(path)
	var match string = ""
	if isMatch && strings.Contains(name, "*") {
		firstIndex := strings.Index(name, "*")
		if firstIndex == 0 {
			match = "^" + strings.Replace(name, "*", ".*", 1)
		}
	}
	log.Info("match" + match)
	var n sync.WaitGroup
	n.Add(1)
	go findFiles(dir, file, &n, match)
	go func() {
		n.Wait()
		close(file)
	}()

	return
}

func findFiles(path string, file chan<- *FileInfo, n *sync.WaitGroup, match string) error {
	defer n.Done()
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, ent := range entries {
		if ent.IsDir() {
			newPath := path + "/" + ent.Name()
			n.Add(1)
			if err := findFiles(newPath, file, n, match); err != nil {
				n.Done()
				return err
			}
			continue
		}
		//正则匹配
		if match != "" {
			if is, _ := regexp.MatchString(match, ent.Name()); !is {
				continue
			}
		}
		file <- &FileInfo{
			File: ent,
			Path: path + "/" + ent.Name(),
		}
	}

	return nil
}

func GetLocalIp() (string, error) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), nil
					}
				}
			}
		}
	}

	return "", nil
}

func ParseFileName(path string) string {
	array := strings.Split(path, "/")
	return array[len(array)-1]
}

//解析文件路径，返回目录和文件名
func ParseFilePath(path string) (string, string) {
	last := strings.LastIndex(path, "/")
	dir := path[0:last]
	name := path[last+1:]

	return dir, name
}
