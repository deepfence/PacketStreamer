IMAGE ?= docker.io/deepfenceio/deepfence_packetstreamer
IMAGE_BUILD ?= docker.io/deepfenceio/deepfence_packetstreamer_build
CC ?= gcc
LDFLAGS ?=
TAGS ?=
STATIC ?= 0
ifeq ($(STATIC),1)
	LDFLAGS += -linkmode external -extldflags "-static"
	TAGS += musl
endif
RELEASE ?= 0
ifeq ($(RELEASE),1)
	LDFLAGS += -s -w
endif

.PHONY: all build docker-bin docker-image test

all: build

build:
	go build -tags '$(TAGS)' --ldflags '$(LDFLAGS)' -o packetstreamer ./main.go

docker-bin: docker-image
	docker cp $(shell docker create --rm $(IMAGE)):/usr/bin/packetstreamer .

docker-image:
	docker build -t $(IMAGE) --build-arg RELEASE=$(RELEASE) .

docker-push:
	docker push $(IMAGE)

docker-test:
	docker build -t $(IMAGE_BUILD) --target builder .
	docker run --rm $(IMAGE_BUILD) make test STATIC=1

test:
	go test -tags '$(TAGS)' ./...
