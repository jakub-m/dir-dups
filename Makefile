bin=bin/listfiles

gofiles=$(shell find . -name \*.go)
gomain=src/cli/listfiles.go

$(bin): $(gofiles)
	go build -o bin $(gomain)

clean:
	rm -fv $(bin)

.PHONY: clean
