BIN="run"
ENTRY="./cmd/run"

all:
	@mkdir -p build
	go build -o build/$(BIN) $(ENTRY)
