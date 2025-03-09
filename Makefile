.PHONY: all build clean fmt lint vet tidy

# Dev database
DB_PATH ?= data/devdatabase.db
WEB_CONTENT_DIR ?= pkg/server/web
DB_SCHEMA_DIR ?= pkg/appdb/sql/schema
DB_MIGRATIONS_DIR ?= pkg/appdb/migrations
# Tool Versions
GOOSE_VERSION ?= v3.11.1
GOLANGCI_LINT_VERSION ?= v1.64.5
SQLC_VERSION ?= v1.28.0

# Tool Variables
LOCALBIN := $(shell pwd)/bin
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)
GOOSE = $(LOCALBIN)/goose-$(GOOSE_VERSION)
SQLC = $(LOCALBIN)/sqlc-$(SQLC_VERSION)

# Ensure bin directory exists
$(shell mkdir -p $(LOCALBIN))

help: # Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo ""
	@awk ' \
		BEGIN {FS = ":.*?# "}; \
		/^##/ { \
			printf "\n\033[1m%s\033[0m\n", substr($$0,4); \
		} \
		/^[a-zA-Z0-9_-]+:.*?# / { \
			printf "  %-20s %s\n", $$1, $$2; \
		}' $(MAKEFILE_LIST)


all: build


## Database Operations
.PHONY: dev-database-schema
database-schema:  sqlite3-install  # Show status of the database tables
	@sqlite3 $(DB_PATH) ".tables"

.PHONY: database-shell
database-shell:  sqlite3-install  # Open the database shell
	@sqlite3 $(DB_PATH)

.PHONY: database-create
database-create:  sqlite3-install  # Populate the database with schema and sample data
	mkdir -p data
	sqlite3 $(DB_PATH) < pkg/appdb/sql/schema/schema.sql

.PHONY: database-destroy
database-destroy:  sqlite3-install  # WARNING This will remove all data
	@echo "WARNING: This will remove all data from the database"
	@rm -f $(DB_PATH)

.PHONY: database-migrate-up
database-migrate-up:  goose-install  # Run all pending migrations
	$(GOOSE) -dir $(DB_MIGRATIONS_DIR) sqlite3 $(DB_PATH) up

.PHONY: database-migrate-down
database-migrate-down:  goose-install  # Rollback the last migration
	$(GOOSE) -dir $(DB_MIGRATIONS_DIR) sqlite3 $(DB_PATH) down

.PHONY: database-migrate-status
database-migrate-status: goose-install   # Show migration status
	$(GOOSE) -dir $(DB_MIGRATIONS_DIR) sqlite3 $(DB_PATH) status

.PHONY: database-migrate-create
database-migrate-create: goose-install   # Create a new migration (usage: make database-migrate-create name=new_migration)
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make database-migrate-create name=new_migration"; \
		exit 1; \
	fi
	$(GOOSE) -dir $(DB_MIGRATIONS_DIR) create $(name) sql


## Development

run:  # Run the application with environment variables
	@if [ -f .env ]; then \
		source .env && go run ./cmd/... serve --debug --webdir $(WEB_CONTENT_DIR); \
	else \
		go run ./cmd/... serve --debug --webdir $(WEB_CONTENT_DIR); \
		exit 1; \
	fi

.PHONY: sqlc  # Generate the SQLC code
sqlc: sqlc-install  # Generate the SQLC code
	@echo "Generating SQLC code... ( $(SQLC) generate )"
	@$(SQLC) generate

build:  # Build the application
	go build -o $(LOCALBIN)/fauxbase ./cmd

clean:  # Remove the application binary
	rm -rf $(LOCALBIN)/bmp

fmt:  # Format the code
	go fmt ./...

.PHONY: lint
lint:  golangci-lint-install  # Lint the code
	$(GOLANGCI_LINT) run ./...

.PHONY: vet
vet:  # Vet the code
	go vet ./...

.PHONY: tidy
tidy:  # Tidy the code
	go mod tidy

.PHONY: test
test:  fmt vet  # Run tests
	go test -v ./...

.PHONY: test-coverage
test-coverage:  # Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out


## Install dev dependencies
.PHONY: sqlite3-install
sqlite3-install:  # Tell you to install sqlite3
	@# find if sqlite3 is installed
	@if ! command -v sqlite3 &> /dev/null; then \
		echo "sqlite3 could not be found"; \
		echo "Please install sqlite3"; \
		echo "  sudo apt-get install sqlite3  # Ubuntu/Debian"; \
		echo "  brew install sqlite3  # MacOS"; \
		echo "  sudo dnf install sqlite3  # Fedora"; \
		echo "  sudo pacman -S sqlite3  # Arch Linux"; \
		echo "  sudo yum install sqlite3  # CentOS/RHEL"; \
		echo "  sudo zypper install sqlite3  # OpenSUSE"; \
		exit 1; \
	fi
	

.PHONY: sqlc-install
sqlc-install:  # Install sqlc 
	@# go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@if [ ! -f "$(SQLC)" ]; then \
		GOBIN=$(LOCALBIN) go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION); \
		mv $(LOCALBIN)/sqlc $(SQLC); \
	fi

.PHONY: goose-install
goose-install:  # Install goose
	@# go install github.com/pressly/goose/v3/cmd/goose@latest
	@if [ ! -f "$(GOOSE)" ]; then \
		GOBIN=$(LOCALBIN) go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION); \
		mv $(LOCALBIN)/goose $(GOOSE); \
	fi


.PHONY: golangci-lint-install
golangci-lint-install:  # Install golangci-lint
	@if [ ! -f "$(GOLANGCI_LINT)" ]; then \
		GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		mv $(LOCALBIN)/golangci-lint $(GOLANGCI_LINT); \
	fi
