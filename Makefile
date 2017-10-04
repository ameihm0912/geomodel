TARGETS = geomodel
GO = go
TESTMMF = $(shell pwd)/GeoIP2-City.mmdb

all: $(TARGETS)

test:
	TESTMMF=$(TESTMMF) $(GO) test -v github.com/ameihm0912/geomodel

geomodel:
	$(GO) install github.com/ameihm0912/geomodel
