LOCATION=github.com/oneshot-uno/oneshot
VERSION=`git describe --tags --abbrev=0`
VERSION_FLAG=$(LOCATION)/cmd.version=$(VERSION)
DATE=`date +"%d-%B-%Y"`
DATE_FLAG=$(LOCATION)/cmd.date="${DATE}"
MANPATH=/usr/local/share/man
PREFIX=/usr/local
HERE=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

oneshot:
	go build -ldflags "-X ${VERSION_FLAG} -X ${DATE_FLAG} -s -w" .

README.md:
	cd doc/md && go run -ldflags "-X ${VERSION_FLAG} -X ${DATE_FLAG}" \
	       	. > $(HERE)/README.md

oneshot.1:
	go run -ldflags "-X ${VERSION_FLAG} -X ${DATE_FLAG}" \
	       	./doc/man/main.go > $(HERE)/oneshot.1

install-man-page: oneshot.1
	mv oneshot.1 $(MANPATH)/man1
	mandb


.PHONY: install
install: oneshot
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	cp $< $(DESTDIR)$(PREFIX)/bin/oneshot

.PHONY: uninstall
uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/oneshot

.PHONY: clean
clean:
	rm -f oneshot


