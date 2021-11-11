binlist=bin/listfiles
binanalyze=bin/analyze
golist=cli/listfiles/main.go
goanalyze=cli/analyze/main.go

gofiles=$(shell find . -name \*.go)

default: $(binlist) $(binanalyze)

$(binlist): $(gofiles)
	go build -o $(binlist) $(golist)

$(binanalyze): $(gofiles)
	go build -o $(binanalyze) $(goanalyze)

test:
	go test ./...

clean:
	rm -fvr bin/

.PHONY: clean test
