SDK	= operator-sdk

REGISTRY      ?= tmaxcloudck
VERSION       ?= 0.0.1

PACKAGE_NAME  = github.com/tmax-cloud/approval-watcher

WATCHER_NAME  = approval-watcher
WATCHER_IMG   = $(REGISTRY)/$(WATCHER_NAME):$(VERSION)

STEP_SERVER_NAME  = approval-step-server
STEP_SERVER_IMG   = $(REGISTRY)/$(STEP_SERVER_NAME):$(VERSION)

BIN = ./build/_output/bin

BUILD_FLAG  = -gcflags all=-trimpath=/home/sunghyun/dev -asmflags all=-trimpath=/home/sunghyun/dev


.PHONY: build build-watcher build-step-server
build: build-watcher build-step-server

build-watcher:
	CGO_ENABLED=0 go build -o $(BIN)/approval-watcher $(BUILD_FLAG) $(PACKAGE_NAME)/cmd/watcher

build-step-server:
	CGO_ENABLED=0 go build -o $(BIN)/approval-step-server $(BUILD_FLAG) $(PACKAGE_NAME)/cmd/step-server


.PHONY: image image-watcher image-step-server
image: image-watcher image-step-server

image-watcher:
	docker build -f build/watcher/Dockerfile -t $(WATCHER_IMG) .

image-step-server:
	docker build -f build/step-server/Dockerfile -t $(STEP_SERVER_IMG) .


.PHONY: push push-watcher push-step-server
push: push-watcher push-step-server

push-watcher:
	docker push $(WATCHER_IMG)

push-step-server:
	docker push $(STEP_SERVER_IMG)
