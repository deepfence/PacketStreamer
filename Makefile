IMAGE ?= docker.io/deepfenceio/deepfence_packetstreamer
CC ?= gcc
LDFLAGS ?=
STATIC ?= 0
ifeq ($(STATIC),1)
	LDFLAGS += -linkmode external -extldflags "-static"
endif
RELEASE ?= 0
ifeq ($(RELEASE),1)
	LDFLAGS += -s -w
endif

.PHONY: all build docker-bin docker-image test

all: build

build:
	go build --ldflags '$(LDFLAGS)' -o packetstreamer ./main.go

docker-bin: docker-image
	docker cp $(shell docker create --rm $(IMAGE)):/usr/bin/packetstreamer .

docker-image:
	docker build -t $(IMAGE) --build-arg RELEASE=$(RELEASE) .

docker-push:
	docker push $(IMAGE)

test:
	go test ./...
