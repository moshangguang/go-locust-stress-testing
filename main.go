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
