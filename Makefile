binlist=bin/listfiles
binanalyze=bin/analyze
golist=src/cli/listfiles/main.go
goanalyze=src/cli/analyze/main.go

gofiles=$(shell find . -name \*.go)

default: $(binlist) $(binanalyze)

$(binlist): $(gofiles)
	go build -o bin $(golist)

$(binanalyze): $(gofiles)
	go build -o bin $(goanalyze)

test:
	go test ./...

clean:
	rm -fv $(binlist) $(binanalyze)

.PHONY: clean test
