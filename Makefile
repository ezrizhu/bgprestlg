.PHONY: clean

all: bgprestlg

# Build executable for Eve program
bgprestlg:
	go mod download
	go build --ldflags "-s -w" -o bin/bgprestlg ./cmd/bgprestlg/


# Build and execute Eve program
start: bgprestlg
	./bin/bgprestlg --log-format pretty

# Format Sojourner source code with Go toolchain
format:
	go mod tidy
	go fmt ./...

# Clean up binary output folder
clean:
	rm -rf bin/
