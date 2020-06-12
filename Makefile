LOCATION="github.com/raphaelreyna/oneshot"
VERSION=`git describe --tags --abbrev=0`
VERSION_FLAG="-X '$(LOCATION)/cmd.version=$(VERSION)'"
MANPATH=/usr/local/share/man
PREFIX=/usr/local

oneshot:
	go build -ldflags $(VERSION_FLAG) .

README.md:
	go run -ldflags $(VERSION_FLAG) \
	       	doc/md/main.go > README.md

install-man-page:
	go run -ldflags $(VERSION_FLAG) \
	       	doc/man/main.go > oneshot.1
	mv oneshot.1 $(MANPATH)/man1


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


