binlist=bin/listfiles
golist=cli/listfiles/main.go

binanalyze=bin/analyze
goanalyze=cli/analyze/main.go

bincleanup=bin/cleanup
gocleanup=cli/cleanup/main.go

gofiles=$(shell find . -name \*.go -or -name \*.gotemplate)

default: test build
build: $(binlist) $(binanalyze) $(bincleanup)

$(binlist): $(gofiles)
	go build -o $(binlist) $(golist)

$(binanalyze): $(gofiles)
	go build -o $(binanalyze) $(goanalyze)

$(bincleanup): $(gofiles)
	go build -o $(bincleanup) $(gocleanup)

test:
	go test ./...

clean:
	rm -fvr bin/

.PHONY: clean test
