UNAME_S = $(shell uname -s)


.PHONY: all
all: test aggregator collector parser

# Start the minimum requirements for the service, i.e. db
.PHONY: up
up:
	docker-compose up -d

# Stop all services
.PHONY: down
down:
	docker-compose down

# Explicitly install dependencies. In most cases this is not required as go will automatically download missing deps.
.PHONY: deps
deps:
	go mod download

.PHONY: build-all aggregator collector parser
build-all: aggregator collector parser

aggregator:
	go  build -mod=readonly -o ./build/aggregator ./cmd/aggregator 

# Build the main executable
collector:
	go  build -mod=readonly -o ./build/collector ./cmd/collector 

# Build the main executable
parser:
	go  build -mod=readonly -o ./build/parser ./cmd/parser 

.PHONY: install-all install-aggregator install-collector install-parser
install-all: install-aggregator install-collector install-parser

install-aggregator:
	go install -mod=readonly ./cmd/aggregator

# Build the main executable
install-collector:
	go install -mod=readonly ./cmd/collector

# Build the main executable
install-parser:
	go install -mod=readonly ./cmd/parser


# This is a specialized build for running the executable inside a minimal scratch container
.PHONY: build-app
build-app:
ifeq (,$(APP_TYPE))
	@echo "provide APP_TYPE"
else
	go build -ldflags="-w -s" -a -o ./main ./cmd/${APP_TYPE}
endif

# Watch for source code changes to recompile + test
.PHONY: watch
watch:
	GO111MODULE=off go get github.com/cortesi/modd/cmd/modd
	modd

# Run all unit tests
.PHONY: test
test:
	go test -short ./...

# Run all benchmarks
.PHONY: bench
bench:
	go test -short -bench=. ./...

# Same as test but with coverage turned on
.PHONY: cover
cover:
	go test -short -cover -covermode=atomic ./...


# Apply https://golang.org/cmd/gofmt/ to all packages
.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: fmt-check
fmt-check:
ifneq ($(shell gofmt -l .),)
	$(error gofmt fail in $(shell gofmt -l .))
endif

# Apply https://github.com/golangci/golangci-lint to changes since forked from master branch
.PHONY: lint
lint:
	golangci-lint run --timeout=5m --enable=unparam --enable=misspell --enable=prealloc --tests=false

# Remove all compiled binaries from the directory
.PHONY: clean
clean:
	go clean

# Analyze the code for any unused dependencies
.PHONY: prune-deps
prune-deps:
	go mod tidy

# Create the service docker image
.PHONY: image
image:
	docker build --force-rm -t dezswap/cosmwasm-etl .

# Migrate database.
.PHONY: parser-migrate-test parser-migrate-up parser-migrate-down parser-generate-migration \
		aggregator-migrate-up aggregator-migrate-down aggregator-generate-migration \
		collector-migrate-up collector-migrate-down collector-generate-migration

parser-migrate-test:
	go test -count=1 -tags=mig,faker ./db/migrations/parser

parser-migrate-up:
	go run db/migrations/parser/main.go

 parser-migrate-down:
	go run db/migrations/parser/main.go down

# Create a new empty migration file.
parser-generate-migration:
	$(eval VERSION := $(shell date +"%Y%m%d%H%M%S"))
	$(eval PATH := db/migrations/parser)
	mkdir -p $(PATH)
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.up.sql
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.down.sql

aggregator-migrate-up:
	go run db/migrations/aggregator/main.go

aggregator-migrate-down:
	go run db/migrations/aggregator/main.go down

# Create a new empty migration file.
aggregator-generate-migration:
	$(eval VERSION := $(shell date +"%Y%m%d%H%M%S"))
	$(eval PATH := db/migrations/aggregator)
	mkdir -p $(PATH)
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.up.sql
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.down.sql


collector-migrate-up:
	go run db/migrations/collector/main.go

collector-migrate-down:
	go run db/migrations/collector/main.go down

# Create a new empty migration file.
collector-generate-migration:
	$(eval VERSION := $(shell date +"%Y%m%d%H%M%S"))
	$(eval PATH := db/migrations/collector)
	mkdir -p $(PATH)
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.up.sql
	cp $(PATH)/template.txt $(PATH)/$(VERSION)_SUMMARY_OF_MIGRATION.down.sql
