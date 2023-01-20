LOCATION=github.com/raphaelreyna/oneshot/v2
VERSION=`git describe --tags --abbrev=0`
VERSION_FLAG=$(LOCATION)/pkg/linkerSetValues.Version=$(VERSION)

oneshot:
	mkdir -p ./build-output
	go build -o ./build-output/oneshot \
		-ldflags "-X ${VERSION_FLAG} -s -w" \
		./cmd/...

oneshot.1:
	go run -ldflags "-X ${VERSION_FLAG}" \
	       	./build-tools/man/main.go > ./build-output/oneshot.1

install-man-page: oneshot.1
	mv ./build-output/oneshot.1 $(MANPATH)/man1
	mandb

.PHONY: itest
itest: oneshot
	go test -count 1 -p 1 -timeout 15s ./integration_testing/...