BIN := api-ustc
INSTALL := $(HOME)/.local/bin/$(BIN)
SERVICE := $(BIN).service

SYSTEMCTL_COMMANDS := start stop restart status reload

.PHONY: all $(SYSTEMCTL_COMMANDS) logs

all: $(BIN) install

$(BIN): $(wildcard *.go) go.sum
	go build -compiler=gccgo -gccgoflags='-s -w' -o "$@"

install: $(INSTALL)

$(INSTALL): $(BIN)
	cp -fp "$<" "$@"

$(SYSTEMCTL_COMMANDS): %:
	systemctl --user $@ $(SERVICE)

logs:
	journalctl --user -eu $(SERVICE)
