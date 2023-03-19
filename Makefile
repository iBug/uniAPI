SRCS := $(shell find . -name "*.go" -type f) go.mod go.sum
BIN := api-ustc
INSTALL := $(HOME)/.local/bin/$(BIN)
SERVICE := $(BIN).service

GOFLAGS := -compiler=gccgo -gccgoflags='-s -w'
#GOFLAGS =

SYSTEMCTL_COMMANDS := start stop restart status reload
JOURNALCTL_COMMANDS := log logs

.PHONY: all $(SYSTEMCTL_COMMANDS) $(JOURNALCTL_COMMANDS)

all: $(BIN) install

$(BIN): $(SRCS)
	go build $(GOFLAGS) -o "$@"

install: $(INSTALL)

$(INSTALL): $(BIN)
	cp -fp "$<" "$@"

$(SYSTEMCTL_COMMANDS): %:
	systemctl --user $@ $(SERVICE)

$(JOURNALCTL_COMMANDS): %:
	journalctl --user -eu $(SERVICE)
