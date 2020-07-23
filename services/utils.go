/*
 * @Author: ybc
 * @Date: 2020-07-22 15:51:25
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-23 17:25:09
 * @Description: file content
 */

package services

import (
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"sync"
)

type FileInfo struct {
	File os.FileInfo
	Path string
}

func PathExists(path string) (os.FileInfo, error) {
	file, err := os.Stat(path)
	if err == nil {
		return nil, err
	}

	if os.IsNotExist(err) {
		return nil, err
	}
	//file == nil说明是目录
	return file, nil
}

func FindFiles(path string, file chan<- *FileInfo, isMatch bool) {
	dir, name := ParseFilePath(path)
	var match string = ""
	if isMatch && strings.Contains(name, "*") {
		firstIndex := strings.Index(name, "*")
		if firstIndex == 0 {
			match = strings.Replace(name, "*", ".*", 1)
		}
	}

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
