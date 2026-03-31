ARTIFACT_NAME := watermeter-image-processor

TESTPARALLELISM := 8

WORKING_DIR := $(shell pwd)

.PHONY: lint
lint::
	golangci-lint run -c .golangci.yml
	go vet ./...

.PHONY: fix
fix::
	golangci-lint fmt -c .golangci.yml

.PHONY: clean
clean::
	rm -r $(WORKING_DIR)/bin

.PHONY: build
build::
	go build -o $(WORKING_DIR)/bin/${ARTIFACT_NAME} ./cmd/processor
	chmod +x $(WORKING_DIR)/bin/${ARTIFACT_NAME}

.PHONY: test
test::
	go test -v -tags=all -parallel ${TESTPARALLELISM} -timeout 2h -covermode atomic -coverprofile=covprofile ./...

.PHONY: coverage
coverage::
	go tool cover -html=covprofile -o coverage.html
	open coverage.html
