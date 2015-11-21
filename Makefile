TARGETS = geomodel
GO = GOPATH=$(shell pwd):$(shell go env GOROOT)/bin go

all: $(TARGETS)

test:
	$(GO) test geomodel

depends:
	$(GO) get github.com/mattbaird/elastigo
	$(GO) get code.google.com/p/gcfg
	$(GO) get github.com/gorilla/context
	$(GO) get github.com/gorilla/mux
	$(GO) get code.google.com/p/go-uuid/uuid
	$(GO) get github.com/jvehent/gozdef
	$(GO) get github.com/oschwald/geoip2-golang

geomodel:
	$(GO) install geomodel

clean:
	rm -f bin/*
	rm -rf pkg/*
