---
title: "flyingdb监控项整理"
---

[TOC]

------

## 1. zabbix_agentd监控脚本运行的流程

### 1.1 在conf目录指定userparameter文件位置

userparameter*指定所有在/home/postgres/zabbix_agent/conf/zabbix_agentd.d/下以userparameter开头的文件

![image-20220814232343315](http://img.sweetbabywow.club/typoara/image-20220814232343315.png)

不同的监控集可以分文件编写

![image-20220814234139376](http://img.sweetbabywow.club/typoara/image-20220814234139376.png)



### 1.2 userparameter用户参数

Zabbix有很多内置的itemkey,但是这些key都是由Zabbix定义好的比较通用的监控项的实现,
如果我们自己想实现某种特有的非通用型的监控项的话,那么我们就得自己去定义数据收集的命令,并且给它指定一个key,
这种机制就叫做User Parameters(用户参数),所以User Parameters的意义就是实现自定义key

- User Parameters只能定义在Agent端,定义在Agent端的zabbix_agent.conf文件中,参数为User Parameters=
- 定义了User Parameters必须重启zabbix-agent服务

语法格式:
UserParameter=<`key`>,<`command`> `无参数`
UserParameter=<`key[*]`>,<`command`> `*表示接受任意个参数,command中可以利用$1,$2,$3...来调用参数,注意awk中对$的引用必须换成$$`

EXAMPLE: `UserParameter可以写在zabbix_agent.conf文件中,也可以写在zabbix_agentd.d目录下``Agent端的Server参数必须允许服务器来采集数据`

![image-20220814234946556](http://img.sweetbabywow.club/typoara/image-20220814234946556.png)



### 1.3 在web端添加监控项示例

这里演示添加pgsql.version监控项的例子

在用户参数中可以看到我们需要在key中使用的参数有

![image-20220815000359838](http://img.sweetbabywow.club/typoara/image-20220815000359838.png)

在对应的pgsql_version.sh脚本中可以看到需要传入的参数其实只有一个，传参的数量可以多但是不能少，多传入参数不影响脚本的运行

![image-20220815000249926](http://img.sweetbabywow.club/typoara/image-20220815000249926.png)

在web端添加监控项

![image-20220815000120050](http://img.sweetbabywow.club/typoara/image-20220815000120050.png)

在主机中添加需要的宏信息

![image-20220815000804830](http://img.sweetbabywow.club/typoara/image-20220815000804830.png)



## 2. zabbix_agentd命令相关参数

### 2.1 概述

Zabbix_agentd是监视各种服务器参数的守护进程，命令描述如下：

```shell
zabbix_agentd [-c config-file]
zabbix_agentd [-c config-file] -p
zabbix_agentd [-c config-file] -t item-key
zabbix_agentd [-c config-file] -R runtime-option
zabbix_agentd -h
zabbix_agentd -V  
```

### 2.2 参数

```shell
-c: 配置配置文件,使用备用配置文件而不是默认配置文件。应该指定绝对路径。
-f: 在前台运行Zabbix代理。
-R runtime-control runtime-option: 根据运行时选项执行管理功能。
-t: 测试项目键,测试单个项目并退出。参见——print获取输出说明。
-h: 帮助,显示此帮助并退出。
-v: 版本,输出版本信息并退出。
-p: 打印,打印已知项目并退出。对于每一项，要么使用通用默认值，要么提供测试的特定默认值。这些默认值作为项目关键参数列在方括号中。返回值用方括号括起来，并以返回值的类型作为前缀，由管道字符分隔。对于用户参数，类型总是t，因为代理不能确定所有可能的返回值。当查询正在运行的代理守护进程时，显示为工作的项目不保证在Zabbix服务器或zabbix_get上工作，因为权限或环境可能不同。返回值类型为:
    d: 有小数部分的数。
    m: 不受支持的。这可能是由于查询仅在活动模式下工作的项(如日志监视项或需要多个收集值的项)造成的。权限问题或不正确的用户参数也可能导致不支持的状态。
    s: 文本。不限制最大长度。
    t: 文本。年代一样。
    u: 无符号整数。
```

运行时控制选项

```shell
log_level_increase[=目标] 增加日志级别，如果没有指定target，将影响所有进程

log_level_decrease[=目标] 降低日志级别，如果没有指定target，将影响所有进程
```

日志级控制目标

```shell
pid：进程标识符
process-type：指定类型的所有进程(例如监听器)
process-type,N：进程类型和编号(例如，监听器，3)
```

