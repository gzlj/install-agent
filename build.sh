#/bin/bash

go env -w  GOPROXY=https://goproxy.io
go env -w  GOPRIVATE=*.goproxy.io,github.com,*.github.com
go env -w  GONOSUMDB=*.goproxy.io,github.com,*.github.com
go env -w  GONOPROXY=none
go env -w  GOSUMDB=off
export GOSUMDB=off
rm -f go.sum
export GOOS=linux
export GOARCH="amd64"
go build ./cmd/agent
