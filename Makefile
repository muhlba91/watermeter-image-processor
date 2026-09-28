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

.PHONY: eval
eval::
	go run ./cmd/eval run $(EVAL_ARGS)

.PHONY: eval-images
eval-images::
	@test -n "$(SRC)" || (echo "usage: make eval-images SRC=<photo.png>" && exit 1)
	go run ./cmd/eval generate -src $(SRC) -accept 667.38 \
		-framing level=1160,505,2014,1561@-26 -framing tilted=1260,605,1814,1361 \
		-variant level:lit:90 -variant level:lit:10 -variant level:dark:30 -variant level:dark:10 \
		-variant tilted:lit:30

.PHONY: coverage
coverage::
	go tool cover -html=covprofile -o coverage.html
	open coverage.html
