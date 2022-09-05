build:
	mkdir -p ./build-output
	go build -o ./build-output/oneshot ./cmd/...

.PHONY: itest
itest: build
	go test -count=1 ./integration_testing/...