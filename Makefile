SRC := src/main.go
BIN := bin/main

build: $(BIN)

$(BIN): $(SRC)
	mkdir -p bin
	go build -o $(BIN) $(SRC)
	chmod +x $(BIN)

run: build
	./$(BIN)

clear:
	rm -f $(BIN)
