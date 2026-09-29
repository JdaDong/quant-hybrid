SHELL := /bin/sh

ROOT := $(CURDIR)
BIN := $(ROOT)/bin
JAVA_BUILD := $(ROOT)/build/java
SCALA_BUILD := $(ROOT)/build/scala

.PHONY: all build build-cpp build-go build-java build-scala test research run smoke clean

all: build

build: build-cpp build-go build-java build-scala

build-cpp:
	mkdir -p $(BIN)
	clang++ -std=c++20 -O2 -Wall -Wextra -pedantic $(ROOT)/cpp/execution_core/main.cpp -o $(BIN)/execution_core

build-go:
	mkdir -p $(BIN)
	go build -trimpath -o $(BIN)/control-plane ./go/control-plane

build-java:
	mkdir -p $(JAVA_BUILD)
	javac -encoding UTF-8 -d $(JAVA_BUILD) $(ROOT)/java/backtest/src/Backtest.java

build-scala:
	mkdir -p $(SCALA_BUILD)
	scalac -encoding UTF-8 -d $(SCALA_BUILD) $(ROOT)/scala/factor/src/main/scala/FactorJob.scala

test: build-go
	go test ./...

research: build-scala build-java
	scala -cp $(SCALA_BUILD) FactorJob $(ROOT)/data/prices.csv $(ROOT)/data/signals.csv
	java -cp $(JAVA_BUILD) Backtest $(ROOT)/data/prices.csv $(ROOT)/data/signals.csv

run: build-cpp build-go
	$(BIN)/control-plane --engine $(BIN)/execution_core --addr 127.0.0.1:8080

smoke: build
	sh $(ROOT)/scripts/smoke.sh

clean:
	rm -rf $(BIN)/* $(JAVA_BUILD)/* $(SCALA_BUILD)/* $(ROOT)/data/signals.csv
