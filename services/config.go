/*
 * @Author: ybc
 * @Date: 2020-06-29 19:30:45
 * @LastEditors: ybc
 * @LastEditTime: 2020-08-12 16:33:17
 * @Description: file content
 */

package services

import (
	"sync"

	"flag"
	"fmt"
	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
	"os"
	"strings"
	"time"
)

var AppConfig *ini.File

var NoticeChan = make(chan *NoticeContent)
var Exit = make(chan int)
var Wait sync.WaitGroup
var FlagOnce sync.Once
var GlobalLock sync.Mutex

type Guard struct {
	Section   *ini.Section
	Config    *Config
	Files     []*FileInfo
	MatchFunc FilterFunc
	Tails     []*tail.Tail
}

type Config struct {
	LogFile          string
	LogDriver        string
	MatchPreg        string
	FilterPreg       string
	NoticeToken      string
	NoticeMobile     string
	NoticeLevel      string
	LogCheckLength   string
	LogSkipLength    string
	LogRecursiveFind bool
}

type NoticeContent struct {
	Path  string
	Line  *tail.Line
	Guard *Guard
}

type FilterFunc func(pattern string, text string) bool

const (
	LOG_DRIVER_ERROR  string = "error"
	LOG_DRIVER_CUSTOM string = "custom"
	DEFAULT_SECTION   string = "DEFAULT"
)

var (
	DEFAULT_CONFIG map[string]string = map[string]string{
		"log_driver":         LOG_DRIVER_ERROR,
		"match_preg":         "(?i)error",
		"filter_preg":        "",
		"notice_level":       "5",
		"log_check_length":   "30",
		"log_skip_length":    "0",
		"notice_mobile":      "",
		"log_recursive_find": "",
	}
	Guards     []*Guard
	ConfigFile *string = flag.String("c", "", "set ini file path")
)

func init() {
	//输出到标准输出（默认是标准错误）
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.JSONFormatter{})
}

func Reload(isReloadConfig bool) {
	GlobalLock.Lock()
	defer GlobalLock.Unlock()
	log.Info("guard restart")
	for _, guard := range Guards {
		if len(guard.Tails) < 1 {
			continue
		}
		for k, tail := range guard.Tails {
			log.Info("stop:", guard.Files[k].Path)
			tail.Stop()
		}
	}
	Guards = Guards[0:0]
	if isReloadConfig {
		InitConfig()
	}
	LoadSections()
}

func InitConfig() bool {
	FlagOnce.Do(func() {
		flag.Parse()
	})
	if !flagUsage() {
		return false
	}
	conf, err := LoadConfig(*ConfigFile)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
	AppConfig = conf

	return true
}

func flagUsage() bool {
	if *ConfigFile != "" {
		return true
	}
	fmt.Fprintf(os.Stdout, `file-guard version: 1.0.0
Usage: file-guard [-c filename] 
Options:

`)
	flag.PrintDefaults()
	return false
}

func Listen() {
	if !InitConfig() {
		return
	}
	LoadSections()
	go HandleNotice()
	go HandelTick()

	<-Exit
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
				LogFile:          config["log_file"],
				LogDriver:        config["log_driver"],
				MatchPreg:        config["match_preg"],
				FilterPreg:       config["filter_preg"],
				NoticeToken:      config["notice_token"],
				NoticeMobile:     config["notice_mobile"],
				NoticeLevel:      config["notice_level"],
				LogCheckLength:   config["log_check_length"],
				LogSkipLength:    config["log_skip_length"],
				LogRecursiveFind: config["log_recursive_find"] == "1",
			},
			MatchFunc: MatchString,
		}
		Guards = append(Guards, guard)
		go guard.Run()
	}
	return
}

func LoadConfig(configFile string) (*ini.File, error) {
	conf, err := ini.Load(configFile)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func HandelTick() {
	t1 := time.Tick(3600 * time.Second)
	for {
		select {
		case <-t1:
			log.Info("tick check")
			Reload(false)
		}
	}
}

func (this *Guard) Run() {
	file, err := PathExists(this.pasePath(this.Config.LogFile))
	if err != nil {
		log.Error("path:", this.Config.LogFile, err.Error())
		return
	}

	var files []*FileInfo
	if file.IsDir() {
		var readFile = make(chan *FileInfo)
		FindFiles(this.Config.LogFile, readFile, this.Config.LogRecursiveFind, true)
		log.Debug("start:", this.Config.LogFile)
		for f := range readFile {
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
	this.listen()
}

func (this *Guard) listen() {
	for _, f := range this.Files {
		go this.tail(f.Path)
	}
}

func (this *Guard) tail(path string) {
	logger := log.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&log.JSONFormatter{})
	config := tail.Config{
		ReOpen:    true,
		Follow:    true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2},
		MustExist: true,
		Poll:      true,
		Logger:    logger,
	}
	t, err := tail.TailFile(path, config)
	if err != nil {
		log.Error(err.Error())
		return
	}
	this.Tails = append(this.Tails, t)
	for line := range t.Lines {
		this.handle(path, line)
	}
	return
}

func (this *Guard) handle(path string, line *tail.Line) {
	if !this.MatchFunc(this.Config.MatchPreg, line.Text) {
		log.Debug("unmatched", line.Text)
		return
	}
	if this.Config.FilterPreg != "" && this.MatchFunc(this.Config.FilterPreg, line.Text) {
		log.Debug("filter", line.Text)
		return
	}
	//send notice
	NoticeChan <- &NoticeContent{
		Line:  line,
		Guard: this,
		Path:  path,
	}
	return
}

//解析文件，兼容*通配符
func (this *Guard) pasePath(path string) string {

	dir, name := ParseFilePath(path)
	if strings.Contains(name, "*") {
		return dir
	}
	return path
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
