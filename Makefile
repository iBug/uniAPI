BIN := api-ustc
INSTALL := $(HOME)/.local/bin/$(BIN)

.PHONY: all restart

all: $(BIN) install

$(BIN): $(wildcard *.go) go.sum
	go build -compiler=gccgo -gccgoflags='-s -w' -o "$@"

install: $(INSTALL)

$(INSTALL): $(BIN)
	cp -fp "$@" "$<"

restart:
	systemctl --user restart api-ustc.service
