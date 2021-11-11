bin=bin/listfiles
gomain=src/cli/listfiles/main.go

gofiles=$(shell find . -name \*.go)

$(bin): $(gofiles)
	go build -o bin $(gomain)

clean:
	rm -fv $(bin)

.PHONY: clean
