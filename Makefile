SDK	= operator-sdk

REGISTRY      ?= tmaxcloudck
VERSION       ?= 0.0.1

PACKAGE_NAME  = github.com/tmax-cloud/approval-watcher

WATCHER_NAME  = approval-watcher
WATCHER_IMG   = $(REGISTRY)/$(WATCHER_NAME):$(VERSION)

STEP_SERVER_NAME  = approval-step-server
STEP_SERVER_IMG   = $(REGISTRY)/$(STEP_SERVER_NAME):$(VERSION)

BIN = ./build/_output/bin


.PHONY: build build-watcher build-step-server
build: build-watcher build-step-server

build-watcher:
	CGO_ENABLED=0 go build -o $(BIN)/approval-watcher $(PACKAGE_NAME)/cmd/watcher

build-step-server:
	CGO_ENABLED=0 go build -o $(BIN)/approval-step-server $(PACKAGE_NAME)/cmd/step-server


.PHONY: gen
gen:
	$(SDK) generate k8s
	$(SDK) generate crds


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


.PHONY: push-latest push-latest-watcher push-latest-step-server
push-latest: push-latest-watcher push-latest-step-server

push-latest-watcher:
	docker tag $(WATCHER_IMG) $(REGISTRY)/$(WATCHER_NAME):latest
	docker push $(REGISTRY)/$(WATCHER_NAME):latest

push-latest-step-server:
	docker tag $(STEP_SERVER_IMG) $(REGISTRY)/$(STEP_SERVER_NAME):latest
	docker push $(REGISTRY)/$(STEP_SERVER_NAME):latest


.PHONY: test test-gen save-sha-gen compare-sha-gen test-verify save-sha-mod compare-sha-mod verify test-unit test-lint
test: test-gen test-verify test-unit test-lint

test-gen: save-sha-gen gen compare-sha-gen

save-sha-gen:
	$(eval CRDSHA=$(shell sha512sum deploy/crds/tmax.io_approvals_crd.yaml))
	$(eval GENSHA=$(shell sha512sum pkg/apis/tmax/v1/zz_generated.deepcopy.go))

compare-sha-gen:
	$(eval CRDSHA_AFTER=$(shell sha512sum deploy/crds/tmax.io_approvals_crd.yaml))
	$(eval GENSHA_AFTER=$(shell sha512sum pkg/apis/tmax/v1/zz_generated.deepcopy.go))
	@if [ "${CRDSHA_AFTER}" = "${CRDSHA}" ]; then echo "deploy/crds/tmax.io_approvals_crd.yaml is not changed"; else echo "deploy/crds/tmax.io_approvals_crd.yaml file is changed"; exit 1; fi
	@if [ "${GENSHA_AFTER}" = "${GENSHA}" ]; then echo "zz_generated.deepcopy.go is not changed"; else echo "zz_generated.deepcopy.go file is changed"; exit 1; fi

test-verify: save-sha-mod verify compare-sha-mod

save-sha-mod:
	$(eval MODSHA=$(shell sha512sum go.mod))
	$(eval SUMSHA=$(shell sha512sum go.sum))

verify:
	go mod verify

compare-sha-mod:
	$(eval MODSHA_AFTER=$(shell sha512sum go.mod))
	$(eval SUMSHA_AFTER=$(shell sha512sum go.sum))
	@if [ "${MODSHA_AFTER}" = "${MODSHA}" ]; then echo "go.mod is not changed"; else echo "go.mod file is changed"; exit 1; fi
	@if [ "${SUMSHA_AFTER}" = "${SUMSHA}" ]; then echo "go.sum is not changed"; else echo "go.sum file is changed"; exit 1; fi

test-unit:
	go test -v ./pkg/...

test-lint:
	golangci-lint run ./... -v -E gofmt --timeout 1h0m0s
