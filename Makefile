TARGETS = geomodel
GO = go
TESTMMF = $(shell pwd)/GeoIP2-City.mmdb

all: $(TARGETS)

test:
	TESTMMF=$(TESTMMF) $(GO) test -v geomodel

depends:
	$(GO) get github.com/mattbaird/elastigo
	$(GO) get code.google.com/p/gcfg
	$(GO) get github.com/gorilla/context
	$(GO) get github.com/gorilla/mux
	$(GO) get code.google.com/p/go-uuid/uuid
	$(GO) get github.com/jvehent/gozdef
	$(GO) get github.com/oschwald/geoip2-golang

geomodel:
	$(GO) install github.com/ameihm0912/geomodel

clean:
	rm -f bin/*
	rm -rf pkg/*
