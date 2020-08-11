# file-guard
实时监控文件变化，匹配符合预期的数据并通知。

### 应用场景
- 错误日志监控，实时捕捉系统错误
- 捕捉日志中关键数据，及时通知

### 特性
- 多项目配置
- 自定义正则匹配、过滤器
- 可控制上报频率
- 钉钉自定义器机器人通知

### 管理命令

#### 启动
```
file-guard -c [配置文件]
```

#### 热重载
```
#发送 USR1信号
kill -USR1 [PID]
```
> 重新加载配置文件，重新扫描监控文件

### 通知级别

各模块相同日志内容通知限制，每个级别对应不同的预警级别，具体如下

| 级别        | 通知间隔   |  通知次数  |
| --------   | -----:  | :----:  |
| 1        | 10秒   |   一小时最多1800次     |
| 2        |   60秒   |   一小时最多60次   |
| 3        |    600秒    |  一小时最多5次  |
| 4        |    半小时    |  一天内最多40次  |
| 5        |    1小时    |  一天内最多24次  |
| 6        |    2小时    |  一天内最多10次  |
| 7        |    4小时    | 一天内最多5次  |
| 8        |    一天    |  一天内最多1次  |


### 配置文件

```
;这里是全局配置，模块配置优先级 > 全局
notice_level = 3

[project1name]
log_file = E:/testlog/dir/*.txt
match_preg = "(?i)success"
;filter_preg = 
notice_token = 

[project2name]
;监控的文件，支持*匹配
log_file = E:/testlog/dir/laravel-*.log

;是否递归查找 1递归查找 0不递归 默认为0
;log_recursive_find = 0

;正则匹配规则，匹配则通知
;match_preg = "(?i)error"

;过滤规则（正则），符合过滤规则的不会通知
; filter_preg = ""

;钉钉通知token
notice_token = 

;通知@人的手机号
;notice_mobile = 

;通知级别
;notice_level = 5

;日志识别最大长度，超出部分不识别
log_check_length = 50

;跳过识别前N个字符，默认为0
log_skip_length = 21
```

## 性能测试


### 测试机器配置
- cpu : 双核 Intel(R) Xeon(R) CPU E5-2660 0 @ 2.20GHz
- 内存：2G


### 测试基准
每10ms写入269字节大小数据到监控文件中。

#### 测试脚本

```
[root@localhost default]# cat file_test.sh
#!/bin/sh
while true
do
        echo "#0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)#0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)"\n >> /tmp/test.log
        sleep 0.01
done
```

#### 监控
```
top - 21:05:55 up 35 days, 11:29,  5 users,  load average: 0.38, 0.16, 0.09
Tasks: 199 total,   1 running, 198 sleeping,   0 stopped,   0 zombie
%Cpu(s):  3.2 us,  8.1 sy,  0.0 ni, 88.7 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
KiB Mem :  1882408 total,   608748 free,   461156 used,   812504 buff/cache
KiB Swap:  2097148 total,  1782428 free,   314720 used.  1145480 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                                                                             
 4533 root      20   0  113308   1536   1240 S   5.3  0.1   0:02.19 file_test.sh                                                                                                                                                        
28585 root      20   0  109280   8968   3728 S   1.0  0.5   0:01.64 file-guard 
```
#### 测试结果
从上面可以看到：file-guard（监控脚本）cpu占用率平均在1%左右。

## 钉钉自定义机器人配置
![image](https://github.com/xuanwolei/file-guard/blob/master/doc/images/rebot_config.png)
> 通知关键词“项目”

## 其他说明
- 目前通知方式只支持钉钉机器人
- 暂未支持机器加签安全设置
- 建议使用**supervisord** 方式启动

## 通知示例
![image](https://github.com/xuanwolei/file-guard/blob/master/doc/images/notice_format.png)