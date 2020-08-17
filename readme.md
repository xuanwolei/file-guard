# file-guard
文件哨兵, 实时监控文件变化，匹配数据并报警。

### 应用场景
- 错误日志监控，实时捕捉系统错误
- 捕捉日志中关键业务数据，实时通知

### 特性
- 多项目配置
- 项目维度的自动重载
- 自定义匹配规则、过滤器
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
> 重新加载配置文件

#### 退出
```
kill -TERM [PID]
```

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

;定时重载开关：1开启，0关闭， 开启后会定时扫描监控文件并重新加载。
auto_reload = 1
;定时重载间隔（秒）
auto_reload_interval = 10
```
#### 配置说明
- 所有配置项都支持全局/项目配置
- 配置优先级：项目配置>全局配置

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

## 性能测试


### 测试机器配置
- cpu : 双核 Intel(R) Xeon(R) CPU E5-2660 0 @ 2.20GHz
- 内存：2G


### 写入不报警日志

#### 测试基准
每5ms写入269字节数据到监控文件中，数据不触发报警。

#### 测试脚本

```
[root@localhost default]# cat file_test.sh
#!/bin/sh
while true
do
        echo "#0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)#0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)"\n >> /tmp/test.log
        sleep 0.005
done
```

#### 查看性能
```
[root@localhost ~]# top
top - 13:47:36 up 37 days,  4:11,  3 users,  load average: 0.13, 0.08, 0.06
Tasks: 196 total,   1 running, 195 sleeping,   0 stopped,   0 zombie
%Cpu(s):  5.8 us, 13.2 sy,  0.0 ni, 80.8 id,  0.0 wa,  0.0 hi,  0.2 si,  0.0 st
KiB Mem :  1882408 total,   348652 free,   696768 used,   836988 buff/cache
KiB Swap:  2097148 total,  1799836 free,   297312 used.   881464 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                                                                             
 6661 root      20   0  113308   1648   1240 S   9.6  0.1   0:04.78 file_test.sh                                                                                                                                                        
20986 root      20   0  109292   9980   4000 S   1.0  0.5   0:13.91 file-guard     file-guard 
```
#### 测试结果
从上面可以看到：file-guard（监控脚本）cpu占用率平均在1%左右。


### 写入报警日志
#### 测试基准

每5ms写入298字节到监控文件中，并且数据触发报警。
#### 写入脚本
```
[root@localhost default]# cat file_test.sh 
#!/bin/sh
while true
do
        echo "errorsi0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)#0 /home/wwwroot/default/storyMarketing/app/Comps/Models/LuckDrawTrait.php(162): App\\Models\\LuckDraw->_addIntegralPrize(1, 446, Array)"\n >> /tmp/test.log
        sleep 0.005
done
```

#### 查看性能

```
top - 11:34:58 up 37 days,  1:58,  3 users,  load average: 0.13, 0.16, 0.11
Tasks: 195 total,   1 running, 194 sleeping,   0 stopped,   0 zombie
%Cpu(s):  7.3 us, 14.0 sy,  0.0 ni, 78.5 id,  0.0 wa,  0.0 hi,  0.2 si,  0.0 st
KiB Mem :  1882408 total,   352896 free,   696304 used,   833208 buff/cache
KiB Swap:  2097148 total,  1799836 free,   297312 used.   881920 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND                                                                                                                                                             
21000 root      20   0  113308   1660   1240 S  10.6  0.1   0:04.94 file_test.sh                                                                                                                                                        
20986 root      20   0  109292   9556   3920 S   3.6  0.5   0:02.13 file-guard  
```

#### 结果
file-guard CPU占用率在3.6-4%左右浮动。

## 注意事项
- 目前通知方式只支持钉钉机器人
- 暂未支持机器加签安全设置
- **USR1**信号在Linux系统支持，windows环境不可用
- 建议使用**supervisord** 方式启动

## 钉钉自定义机器人配置
![image](https://note.youdao.com/yws/public/resource/637e8ee93a9c3d09cafbea79599c657a/xmlnote/A283924347694E30BCEA087A9FAF1F6E/10664)
> 通知关键词“项目”

## 通知示例
![image](https://note.youdao.com/yws/public/resource/637e8ee93a9c3d09cafbea79599c657a/xmlnote/582DE12344084FB48A5BC46DD0C02B62/10662)







