bin=bin/listfiles

gofiles=$(shell find . -name \*.go)
gomain=src/listfiles.go

$(bin): $(gofiles)
	go build -o bin $(gomain)

