# go-locust-stress-testing

## 背景介绍

locust是一款用Python编写的压测工具，可以模拟成千上百的用户对我们的服务进行压测，并生成各种协助我们评估服务性能的压测指标和报表；而golang作为主流开发语言，很多golang开发者经常面临各种各样的压测需求。

go-locust-stress-testing压测框架集成了locust、golang、docker，开发者只要专注编写压测的golang代码，再以docker容器的方式进行部署，locust即会拉起golang压测代码进行压测，生成各种压测指标和报表。

## golang压测代码

假设我们想压测下面两个接口，我们在下面代码的<1>和<2>处两个方法，会各创建一个Task对象来压测两个接口，<1>、<2>两个方法的实现都大同小异，都是计算一次网络请求的消耗时间，如果网络请求出错，在<3>处传入耗时和报错原因。如果http的响应码不是200，则在<4>处传入耗时和响应码，响应码为200则在<5>处传入耗时和返回字节长度。

- https://mock.api7.ai/
- https://httpbin.org/

```go
package main

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/myzhan/boomer"
)

const HttpRequestType = "http"

func buildTestingMockApiTask() *boomer.Task {
	taskName := "压测MockApi"
	return &boomer.Task{
		Weight: 1,
		Fn: func() {
			request, _ := http.NewRequest(http.MethodGet, "https://mock.api7.ai/", bytes.NewBuffer(nil))
			startTime := time.Now()
			response, err := http.DefaultClient.Do(request)
			elapsed := time.Since(startTime)
			if err != nil {
				boomer.RecordFailure(HttpRequestType, taskName, elapsed.Milliseconds(), err.Error()) //<3>
			} else {
				if response.Body != nil {
					defer response.Body.Close()
				}
				length := response.ContentLength
				if response.StatusCode != http.StatusOK {
					boomer.RecordFailure(HttpRequestType, taskName, elapsed.Milliseconds(), fmt.Sprintf("statusCode:%d", response.StatusCode)) //<4>
				} else {
					boomer.RecordSuccess(HttpRequestType, taskName, elapsed.Milliseconds(), length) //<5>
				}
			}

		},
		Name: taskName,
	}
}
func buildTestingHttpBinTask() *boomer.Task {
	taskName := "压测HttpBin"
	return &boomer.Task{
		Weight: 1,
		Fn: func() {
			request, _ := http.NewRequest(http.MethodGet, "https://httpbin.org/", bytes.NewBuffer(nil))
			startTime := time.Now()
			response, err := http.DefaultClient.Do(request)
			elapsed := time.Since(startTime)
			if err != nil {
				boomer.RecordFailure(HttpRequestType, taskName, elapsed.Milliseconds(), err.Error())
			} else {
				if response.Body != nil {
					defer response.Body.Close()
				}
				length := response.ContentLength
				if response.StatusCode != http.StatusOK {
					boomer.RecordFailure(HttpRequestType, taskName, elapsed.Milliseconds(), fmt.Sprintf("statusCode:%d", response.StatusCode))
				} else {
					boomer.RecordSuccess(HttpRequestType, taskName, elapsed.Milliseconds(), length)
				}
			}

		},
		Name: taskName,
	}
}
func main() {
	taskList := []*boomer.Task{
		buildTestingMockApiTask(), //<1>压测MockApi
		buildTestingHttpBinTask(), //<2>压测HttpBin
	}
	boomer.Run(taskList...)
}

```



## 运行部署

编写完压测代码后，我们就可以构建镜像并运行，locust默认的web端口是8090，我们这里也使用宿主机8090端口映射到容器内8090端口。

```bash
# 构建镜像
docker build -t go-locust-stress-testing:latest .
# 运行容器
docker run -d -p 8090:8090 go-locust-stress-testing
```

容器启动后，我们就可以使用http://{host}:8090/ 来访问locust主节点，如果是本机，则访问：http://localhost:8090/ 。进入locust web页面后，设置用户数、用户产生速率，locust就可以拉起我们之前golang编写的压测代码。



## 运行原理

如果想进一步了解go-locust-stress-testing框架的运行原理，可以看下这一部分，如果对go-locust-stress-testing的要求，仅仅在使用阶段，可以只看到上面的【运行部署】。

首先是golang部分，是使用github.com/myzhan/boomer 将压测的耗时、字节长度上报给locust主节点。因此，我们在打包的第一步，要先将我们的main.go文件，打包成一个可执行程序。

下面的Dockerfile脚本非常容易理解，基于golang:1.22.10这个容器，执行build.sh脚本。

```dockerfile
FROM golang:1.22.10 AS golang-builder
LABEL authors="lf"

WORKDIR /app
COPY . .

RUN ["/bin/bash","build.sh"]
```



build.sh脚本的内容也很简单，执行go mod tidy命令拉取项目所需要的库，生成可执行程序main文件。

```bash
export GO111MODULE=on; export GOPROXY=https://goproxy.cn,direct; go mod tidy
export GO111MODULE=on; export GOPROXY=https://goproxy.cn,direct; go get github.com/myzhan/boomer@master
go build -o main main.go
```



生成main文件后，我们不单单要把main程序启动起来，还要启动locust主节点。所以，这里我们要使用supervisor来管理go程序和locust程序的启动和监控。

supervisor是一个进程管理服务，主要用来将运行在前台的进程转为后台运行，并实时监控进程的状态。当程序出现异常或崩溃时，supervisor会自动将该进程拉起。

下面的dockerfile脚本，我们基于python:3.11.11这个容器，安装supervisor进程管理服务，之后我们拷贝Dockerfile第一个阶段生成的main文件、和项目原有的master.py、requirements.txt、supervisord.conf文件

requirements.txt是我们依赖的python库，目前我们仅依赖locust。下面我们主要来看看supervisord.conf文件，这是运行golang压测程序和locust主节点的关键。当我们运行/usr/bin/supervisord，supervisord程序会去读取supervisord.conf文件，根据文件的配置启动并监控程序。

```dockerfile
FROM python:3.11.11 
RUN apt-get update && \
    apt-get install -y cron supervisor && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=golang-builder /app/main .
COPY --from=golang-builder /app/master.py .
COPY --from=golang-builder /app/requirements.txt .
COPY --from=golang-builder /app/supervisord.conf /etc/supervisor/conf.d/supervisord.conf
RUN pip install --upgrade pip
RUN pip install -r /app/requirements.txt

CMD ["/usr/bin/supervisord"]
```



supervisord.conf包含两个程序：locust和go-testing

- command就是启动程序的命令。
- autostart=true代表当supervisord启动时自动启动该进程。
- autorestart=true代表进程异常退出后自动重启。
- stdout_logfile是标准输出日志文件路径。
- stderr_logfile是标准错误输出日志的路径。

```
[supervisord]
nodaemon=true

[program:locust]
command=locust -f /app/master.py --master --web-port=8090
autostart=true
autorestart=true
stdout_logfile=/app/locust.log
stderr_logfile=/app/locust.err.log


[program:go-testing]
command=/app/main
autostart=true
autorestart=true
stdout_logfile=/app/go-testing.log
stderr_logfile=/app/go-testing.err.log
```

