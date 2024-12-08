export GO111MODULE=on; export GOPROXY=https://goproxy.cn,direct; go mod tidy
export GO111MODULE=on; export GOPROXY=https://goproxy.cn,direct; go get github.com/myzhan/boomer@master
go build -o main main.go