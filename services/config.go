/*
 * @Author: ybc
 * @Date: 2020-06-29 19:30:45
 * @LastEditors: ybc
 * @LastEditTime: 2020-07-22 19:51:39
 * @Description: file content
 */

package services

import (
	"fmt"
	"sync"

	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var AppConfig *ini.File

var NoticeChan = make(chan *NoticeContent)
var Exit = make(chan int)
var Wait sync.WaitGroup

type Guard struct {
	Section *ini.Section
	Config  *Config
	Files   []*FileInfo
}

type Config struct {
	LogFile     string
	LogDriver   string
	FilterPreg  string
	NoticeToken string
	NoticeLevel string
}

type NoticeContent struct {
	Message string
	Guard   *Guard
}

var (
	DEFAULT_CONFIG map[string]string = map[string]string{
		"log_driver":  "laravel",
		"filter_preg": "",
		"NoticeLevel": "1",
	}
)

func init() {
	conf, err := LoadConfig("./conf/app.ini")
	if err != nil {
		panic(err)
	}
	AppConfig = conf
	// s := conf.Section("dev")
	// if s == nil{
	// 	fmt.Println("not nil")
	// }
	// fmt.Println(s.Key("a").String())
}

func LoadConfig(configFile string) (*ini.File, error) {
	conf, err := ini.Load(configFile)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func LoadSections() {
	defaultSection := AppConfig.Section("DEFAULT")
	defaultConfig := defaultSection.KeysHash()
	sections := AppConfig.Sections()
	for _, section := range sections {
		if section.Name() == "DEFAULT" {
			continue
		}
		hashConfig := section.KeysHash()
		config := StringMapSetDefaultVal(hashConfig, defaultConfig)
		guard := &Guard{
			Section: section,
			Config: &Config{
				LogFile:     config["log_file"],
				LogDriver:   config["log_driver"],
				FilterPreg:  config["filter_preg"],
				NoticeToken: config["notice_token"],
				NoticeLevel: config["notice_level"],
			},
		}
		go guard.Run()
	}

	go handleNotice()

	<-Exit
}

func StringMapSetDefaultVal(hash map[string]string, defaultHash map[string]string) map[string]string {
	for k, v := range defaultHash {
		if hash[k] != "" {
			continue
		} else if v == "" {
			hash[k] = DEFAULT_CONFIG[k]
		}
		hash[k] = v
	}

	return hash
}

func handleNotice() {
	for {
		select {
		case notice := <-NoticeChan:
			fmt.Println("receive:", notice)
		}
	}
}

func (this *Guard) Run() {
	fmt.Println("log_file", this.Config.LogFile)
	file, err := PathExists(this.Config.LogFile)
	if err != nil {
		log.Error("path:", this.Config.LogFile, err.Error())
		return
	}

	var files []*FileInfo
	if file == nil {
		var readFile = make(chan *FileInfo)
		FindFiles(this.Config.LogFile, readFile, true)
		log.Debug("start:", this.Config.LogFile)
		for f := range readFile {
			log.Info("file:", f.Path)
			files = append(files, f)
		}
		log.Debug("finish")
	} else {
		files = append(files, &FileInfo{
			File: file,
			Path: this.Config.LogFile,
		})
	}

	this.Files = files
	this.filter()
	NoticeChan <- &NoticeContent{
		Guard:   this,
		Message: "test",
	}
}

func (this *Guard) filter() {
	for _, f := range this.Files {
		go this.tail(f.Path)
	}
}

func (this *Guard) tail(path string) {
	config := tail.Config{
		ReOpen:    true,                                 // 重新打开
		Follow:    true,                                 // 是否跟随
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // 从文件的哪个地方开始读
		MustExist: false,                                // 文件不存在不报错
		Poll:      true,
	}
	t, err := tail.TailFile(path, config)
	if err != nil {
		log.Error(err.Error())
		return
	}
	for line := range t.Lines {
		this.handle(path, line)
	}
	return
}

func (this *Guard) handle(path string, line *tail.Line) {
	fmt.Println("文件：", path, "变化：", line.Text)
}
