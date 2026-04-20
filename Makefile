BINARY := calendarCli
SRC    := ./cmd/main.go

.PHONY: build run fmt vet clean

build:
	go build -o $(BINARY) $(SRC)

run: build
	./$(BINARY)

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
