BUILD := `git rev-parse --short HEAD`

.PHONY: build

build:
	go build -o ./build/teler \
		-ldflags "-X 'github.com/wcgcyx/teler/rpc.Debug=$(DEBUG)' \
		-X 'github.com/wcgcyx/teler/version.Version=$(BUILD)'" \
		cmd/teler/main.go

utest:
	go test -v --count=1 ./...

clean:
	rm -rf ./build/*