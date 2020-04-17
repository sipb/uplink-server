.PHONY: build package run stop run-client run-server stop-client stop-server restart restart-server restart-client start-docker clean-dist clean nuke check-style check-client-style check-server-style check-unit-tests test dist prepare-enteprise run-client-tests setup-run-client-tests cleanup-run-client-tests test-client build-linux build-osx build-windows internal-test-web-client vet run-server-for-web-client-tests diff-config

ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

IS_CI ?= false
MM_NO_DOCKER ?= false
# Build Flags
BUILD_NUMBER ?= $(BUILD_NUMBER:)
BUILD_DATE = $(shell date -u)
BUILD_HASH = $(shell git rev-parse HEAD)
# If we don't set the build number it defaults to dev
ifeq ($(BUILD_NUMBER),)
	BUILD_NUMBER := dev
endif
BUILD_ENTERPRISE = true
BUILD_ENTERPRISE_READY = true
BUILD_TYPE_NAME = enterprise
BUILD_HASH_ENTERPRISE = not-applicable
LDAP_DATA ?= test
BUILD_WEBAPP_DIR ?= ../uplink-webapp
BUILD_CLIENT = false
BUILD_HASH_CLIENT = independant
ifneq ($(wildcard $(BUILD_WEBAPP_DIR)/.),)
	ifeq ($(BUILD_CLIENT),true)
		BUILD_CLIENT = true
		BUILD_HASH_CLIENT = $(shell cd $(BUILD_WEBAPP_DIR) && git rev-parse HEAD)
	else
		BUILD_CLIENT = false
	endif
else
	BUILD_CLIENT = false
endif

# Golang Flags
GOPATH ?= $(shell go env GOPATH)
GOFLAGS ?= $(GOFLAGS:) -mod=vendor
GO=go
DELVE=dlv
LDFLAGS += -X "github.com/mattermost/mattermost-server/model.BuildNumber=$(BUILD_NUMBER)"
LDFLAGS += -X "github.com/mattermost/mattermost-server/model.BuildDate=$(BUILD_DATE)"
LDFLAGS += -X "github.com/mattermost/mattermost-server/model.BuildHash=$(BUILD_HASH)"
LDFLAGS += -X "github.com/mattermost/mattermost-server/model.BuildHashEnterprise=$(BUILD_HASH_ENTERPRISE)"
LDFLAGS += -X "github.com/mattermost/mattermost-server/model.BuildEnterpriseReady=$(BUILD_ENTERPRISE_READY)"
GO_MAJOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_MINOR_VERSION = $(shell $(GO) version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
MINIMUM_SUPPORTED_GO_MAJOR_VERSION = 1
MINIMUM_SUPPORTED_GO_MINOR_VERSION = 13
GO_VERSION_VALIDATION_ERR_MSG = Golang version is not supported, please update to at least $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION).$(MINIMUM_SUPPORTED_GO_MINOR_VERSION)

# GOOS/GOARCH of the build host, used to determine whether we're cross-compiling or not
BUILDER_GOOS_GOARCH="$(shell $(GO) env GOOS)_$(shell $(GO) env GOARCH)"

PLATFORM_FILES="./cmd/mattermost/main.go"

# Output paths
DIST_ROOT=dist
DIST_PATH=$(DIST_ROOT)/mattermost

# Tests
TESTS=.

TESTFLAGS ?= -short
TESTFLAGSEE ?= -short

# Packages lists
TE_PACKAGES=$(shell $(GO) list ./...)

# Plugins Packages
PLUGIN_PACKAGES?=
#PLUGIN_PACKAGES += mattermost-plugin-zoom-v1.1.2
#PLUGIN_PACKAGES += mattermost-plugin-autolink-v1.1.1
#PLUGIN_PACKAGES += mattermost-plugin-nps-v1.0.3
#PLUGIN_PACKAGES += mattermost-plugin-custom-attributes-v1.0.2
#PLUGIN_PACKAGES += mattermost-plugin-github-v0.11.0
#PLUGIN_PACKAGES += mattermost-plugin-welcomebot-v1.1.1
#PLUGIN_PACKAGES += mattermost-plugin-aws-SNS-v1.0.2
#PLUGIN_PACKAGES += mattermost-plugin-antivirus-v0.1.1
#PLUGIN_PACKAGES += mattermost-plugin-jira-v2.2.2
#PLUGIN_PACKAGES += mattermost-plugin-gitlab-v1.0.1
#PLUGIN_PACKAGES += mattermost-plugin-jenkins-v1.0.0

# Prepares the enterprise build if exists. The IGNORE stuff is a hack to get the Makefile to execute the commands outside a target
ifeq ($(BUILD_ENTERPRISE_READY),true)
	IGNORE:=$(shell echo Enterprise build selected, preparing)
else
	IGNORE:=$(shell rm -f imports/imports.go)
endif

ALL_PACKAGES=$(TE_PACKAGES)


all: run ## Alias for 'run'.

include build/*.mk

start-docker: ## Starts the docker containers for local development.
ifneq ($(IS_CI),false)
	@echo CI Build: skipping docker start
else ifeq ($(MM_NO_DOCKER),true)
	@echo No Docker Enabled: skipping docker start
else
	@echo Starting docker containers

	docker-compose run --rm start_dependencies
	cat tests/${LDAP_DATA}-data.ldif | docker-compose exec -T openldap bash -c 'ldapadd -x -D "cn=admin,dc=mm,dc=test,dc=com" -w mostest || true';
endif

stop-docker: ## Stops the docker containers for local development.
ifeq ($(MM_NO_DOCKER),true)
	@echo No Docker Enabled: skipping docker stop
else
	@echo Stopping docker containers

	docker-compose stop
endif

clean-docker: ## Deletes the docker containers for local development.
ifeq ($(MM_NO_DOCKER),true)
	@echo No Docker Enabled: skipping docker clean
else
	@echo Removing docker containers

	docker-compose down -v
	docker-compose rm -v
endif


plugin-checker:
	$(GO) run $(GOFLAGS) ./plugin/checker

golangci-lint: ## Run golangci-lint on codebase
# https://stackoverflow.com/a/677212/1027058 (check if a command exists or not)
	@if ! [ -x "$$(command -v golangci-lint)" ]; then \
		echo "golangci-lint is not installed. Please see https://github.com/golangci/golangci-lint#install for installation instructions."; \
		exit 1; \
	fi; \

	@echo Running golangci-lint
	golangci-lint run ./...
ifeq ($(BUILD_ENTERPRISE_READY),true)
	golangci-lint run ./enterprise/...
endif

i18n-extract: ## Extract strings for translation from the source code
	env GO111MODULE=off $(GO) get -u github.com/mattermost/mattermost-utilities/mmgotool
	$(GOPATH)/bin/mmgotool i18n extract

store-mocks: ## Creates mock files.
	env GO111MODULE=off $(GO) get -u github.com/vektra/mockery/...
	$(GOPATH)/bin/mockery -dir store -all -output store/storetest/mocks -note 'Regenerate this file using `make store-mocks`.'

store-layers: ## Generate layers for the store
	$(GO) generate $(GOFLAGS) ./store

filesstore-mocks: ## Creates mock files.
	env GO111MODULE=off $(GO) get -u github.com/vektra/mockery/...
	$(GOPATH)/bin/mockery -dir services/filesstore -all -output services/filesstore/mocks -note 'Regenerate this file using `make filesstore-mocks`.'

ldap-mocks: ## Creates mock files for ldap.
	env GO111MODULE=off $(GO) get -u github.com/vektra/mockery/...
	$(GOPATH)/bin/mockery -dir enterprise/ldap -all -output enterprise/ldap/mocks -note 'Regenerate this file using `make ldap-mocks`.'

plugin-mocks: ## Creates mock files for plugins.
	env GO111MODULE=off $(GO) get -u github.com/vektra/mockery/...
	$(GOPATH)/bin/mockery -dir plugin -name API -output plugin/plugintest -outpkg plugintest -case underscore -note 'Regenerate this file using `make plugin-mocks`.'
	$(GOPATH)/bin/mockery -dir plugin -name Hooks -output plugin/plugintest -outpkg plugintest -case underscore -note 'Regenerate this file using `make plugin-mocks`.'
	$(GOPATH)/bin/mockery -dir plugin -name Helpers -output plugin/plugintest -outpkg plugintest -case underscore -note 'Regenerate this file using `make plugin-mocks`.'

einterfaces-mocks: ## Creates mock files for einterfaces.
	env GO111MODULE=off $(GO) get -u github.com/vektra/mockery/...
	$(GOPATH)/bin/mockery -dir einterfaces -all -output einterfaces/mocks -note 'Regenerate this file using `make einterfaces-mocks`.'

pluginapi: ## Generates api and hooks glue code for plugins
	$(GO) generate $(GOFLAGS) ./plugin

check-licenses: ## Checks license status.
	./scripts/license-check.sh $(TE_PACKAGES) $(EE_PACKAGES)

check-prereqs: ## Checks prerequisite software status.
	./scripts/prereq-check.sh

check-style: golangci-lint plugin-checker check-licenses check-plugin-golint ## Runs golangci against all packages and also ensures plugin package golint compliant

check-plugin-golint: # Checks if golint returns any uncompliant code for any file that starts with plugin/helpers
	@! golint ./plugin/ | grep plugin/helpers

test-te-race: ## Checks for race conditions in the team edition.
	@echo Testing TE race conditions

	@echo "Packages to test: "$(TE_PACKAGES)

	@for package in $(TE_PACKAGES); do \
		echo "Testing "$$package; \
		$(GO) test $(GOFLAGS) -race -run=$(TESTS) -test.timeout=4000s $$package || exit 1; \
	done

test-server-race: test-te-race test-ee-race ## Checks for race conditions.
	find . -type d -name data -not -path './vendor/*' | xargs rm -rf

do-cover-file: ## Creates the test coverage report file.
	@echo "mode: count" > cover.out

go-junit-report:
	env GO111MODULE=off $(GO) get -u github.com/jstemmer/go-junit-report

test-compile: ## Compile tests.
	@echo COMPILE TESTS

	for package in $(TE_PACKAGES) $(EE_PACKAGES); do \
		$(GO) test $(GOFLAGS) -c $$package; \
	done

test-db-migration: start-docker ## Gets diff of upgrade vs new instance schemas.
	./scripts/mysql-migration-test.sh
	./scripts/psql-migration-test.sh

test-server: start-docker go-junit-report do-cover-file ## Runs tests.
	@echo Running only TE tests
	./scripts/test.sh "$(GO)" "$(GOFLAGS)" "$(ALL_PACKAGES)" "$(TESTS)" "$(TESTFLAGS)"

internal-test-web-client: ## Runs web client tests.
	$(GO) run $(GOFLAGS) $(PLATFORM_FILES) test web_client_tests

run-server-for-web-client-tests: ## Tests the server for web client.
	$(GO) run $(GOFLAGS) $(PLATFORM_FILES) test web_client_tests_server

test-client: ## Test client app.
	@echo Running client tests

	cd $(BUILD_WEBAPP_DIR) && $(MAKE) test

test: test-server test-client ## Runs all checks and tests below (except race detection and postgres).

cover: ## Runs the golang coverage tool. You must run the unit tests first.
	@echo Opening coverage info in browser. If this failed run make test first

	$(GO) tool cover -html=cover.out
	$(GO) tool cover -html=ecover.out

test-data: start-docker ## Add test data to the local instance.
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) config set TeamSettings.MaxUsersPerTeam 100
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) sampledata -w 4 -u 60

	@echo You may need to restart the Mattermost server before using the following
	@echo ========================================================================
	@echo Login with a system admin account username=sysadmin password=Sys@dmin-sample1
	@echo Login with a regular account username=user-1 password=SampleUs@r-1
	@echo ========================================================================

validate-go-version: ## Validates the installed version of go against Mattermost's minimum requirement.
	@if [ $(GO_MAJOR_VERSION) -gt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		exit 0 ;\
	elif [ $(GO_MAJOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MAJOR_VERSION) ]; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	elif [ $(GO_MINOR_VERSION) -lt $(MINIMUM_SUPPORTED_GO_MINOR_VERSION) ] ; then \
		echo '$(GO_VERSION_VALIDATION_ERR_MSG)';\
		exit 1; \
	fi

run-server: validate-go-version start-docker ## Starts the server.
	@echo Running mattermost for development

	mkdir -p $(BUILD_WEBAPP_DIR)/dist/files
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) --disableconfigwatch 2>&1 | \
	    $(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) logs --logrus &

run-server-simple:
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) --disableconfigwatch 2>&1

run-server-only:
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) --disableconfigwatch 2>&1 | \
	    $(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) logs --logrus &

debug-server: start-docker ## Compile and start server using delve.
	mkdir -p $(BUILD_WEBAPP_DIR)/dist/files
	$(DELVE) debug $(PLATFORM_FILES) --build-flags="-ldflags '\
		-X github.com/mattermost/mattermost-server/model.BuildNumber=$(BUILD_NUMBER)\
		-X \"github.com/mattermost/mattermost-server/model.BuildDate=$(BUILD_DATE)\"\
		-X github.com/mattermost/mattermost-server/model.BuildHash=$(BUILD_HASH)\
		-X github.com/mattermost/mattermost-server/model.BuildHashEnterprise=$(BUILD_HASH_ENTERPRISE)\
		-X github.com/mattermost/mattermost-server/model.BuildEnterpriseReady=$(BUILD_ENTERPRISE_READY)'"

debug-server-headless: start-docker ## Debug server from within an IDE like VSCode or IntelliJ.
	mkdir -p $(BUILD_WEBAPP_DIR)/dist/files
	$(DELVE) debug --headless --listen=:2345 --api-version=2 --accept-multiclient $(PLATFORM_FILES) --build-flags="-ldflags '\
		-X github.com/mattermost/mattermost-server/model.BuildNumber=$(BUILD_NUMBER)\
		-X \"github.com/mattermost/mattermost-server/model.BuildDate=$(BUILD_DATE)\"\
		-X github.com/mattermost/mattermost-server/model.BuildHash=$(BUILD_HASH)\
		-X github.com/mattermost/mattermost-server/model.BuildHashEnterprise=$(BUILD_HASH_ENTERPRISE)\
		-X github.com/mattermost/mattermost-server/model.BuildEnterpriseReady=$(BUILD_ENTERPRISE_READY)'"

run-cli: start-docker ## Runs CLI.
	@echo Running mattermost for development
	@echo Example should be like 'make ARGS="-version" run-cli'

	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) ${ARGS}

run-client: ## Runs the webapp.
	@echo Running mattermost client for development

	ln -nfs $(BUILD_WEBAPP_DIR)/dist client
	cd $(BUILD_WEBAPP_DIR) && $(MAKE) run

run-client-fullmap: ## Legacy alias to run-client
	@echo Running mattermost client for development

	cd $(BUILD_WEBAPP_DIR) && $(MAKE) run

run: check-prereqs run-server run-client ## Runs the server and webapp.

run-fullmap: run-server run-client ## Legacy alias to run

stop-server: ## Stops the server.
	@echo Stopping mattermost

ifeq ($(BUILDER_GOOS_GOARCH),"windows_amd64")
	wmic process where "Caption='go.exe' and CommandLine like '%go.exe run%'" call terminate
	wmic process where "Caption='mattermost.exe' and CommandLine like '%go-build%'" call terminate
else
	@for PID in $$(ps -ef | grep "[g]o run" | grep "disableconfigwatch" | awk '{ print $$2 }'); do \
		echo stopping go $$PID; \
		kill $$PID; \
	done
	@for PID in $$(ps -ef | grep "[g]o-build" | grep "disableconfigwatch" | awk '{ print $$2 }'); do \
		echo stopping mattermost $$PID; \
		kill $$PID; \
	done
endif

stop-client: ## Stops the webapp.
	@echo Stopping mattermost client

	cd $(BUILD_WEBAPP_DIR) && $(MAKE) stop

stop: stop-server stop-client stop-docker ## Stops server, client and the docker compose.

restart: restart-server restart-client ## Restarts the server and webapp.

restart-server: | stop-server run-server ## Restarts the mattermost server to pick up development change.

restart-client: | stop-client run-client ## Restarts the webapp.

run-job-server: ## Runs the background job server.
	@echo Running job server for development
	$(GO) run $(GOFLAGS) -ldflags '$(LDFLAGS)' $(PLATFORM_FILES) jobserver --disableconfigwatch &

config-ldap: ## Configures LDAP.
	@echo Setting up configuration for local LDAP

	@sed -i'' -e 's|"LdapServer": ".*"|"LdapServer": "localhost"|g' config/config.json
	@sed -i'' -e 's|"BaseDN": ".*"|"BaseDN": "dc=mm,dc=test,dc=com"|g' config/config.json
	@sed -i'' -e 's|"BindUsername": ".*"|"BindUsername": "cn=admin,dc=mm,dc=test,dc=com"|g' config/config.json
	@sed -i'' -e 's|"BindPassword": ".*"|"BindPassword": "mostest"|g' config/config.json
	@sed -i'' -e 's|"FirstNameAttribute": ".*"|"FirstNameAttribute": "cn"|g' config/config.json
	@sed -i'' -e 's|"LastNameAttribute": ".*"|"LastNameAttribute": "sn"|g' config/config.json
	@sed -i'' -e 's|"NicknameAttribute": ".*"|"NicknameAttribute": "cn"|g' config/config.json
	@sed -i'' -e 's|"EmailAttribute": ".*"|"EmailAttribute": "mail"|g' config/config.json
	@sed -i'' -e 's|"UsernameAttribute": ".*"|"UsernameAttribute": "uid"|g' config/config.json
	@sed -i'' -e 's|"IdAttribute": ".*"|"IdAttribute": "uid"|g' config/config.json
	@sed -i'' -e 's|"LoginIdAttribute": ".*"|"LoginIdAttribute": "uid"|g' config/config.json
	@sed -i'' -e 's|"GroupDisplayNameAttribute": ".*"|"GroupDisplayNameAttribute": "cn"|g' config/config.json
	@sed -i'' -e 's|"GroupIdAttribute": ".*"|"GroupIdAttribute": "entryUUID"|g' config/config.json

config-reset: ## Resets the config/config.json file to the default.
	@echo Resetting configuration to default
	rm -f config/config.json
	OUTPUT_CONFIG=$(PWD)/config/config.json $(GO) generate $(GOFLAGS) ./config

diff-config: ## Compares default configuration between two mattermost versions
	@./scripts/diff-config.sh

clean: stop-docker ## Clean up everything except persistant server data.
	@echo Cleaning

	rm -Rf $(DIST_ROOT)
	$(GO) clean $(GOFLAGS) -i ./...

	cd $(BUILD_WEBAPP_DIR) && $(MAKE) clean

	find . -type d -name data -not -path './vendor/*' | xargs rm -rf
	rm -rf logs

	rm -f mattermost.log
	rm -f mattermost.log.jsonl
	rm -f npm-debug.log
	rm -f .prepare-go
	rm -f enterprise
	rm -f cover.out
	rm -f ecover.out
	rm -f *.out
	rm -f *.test
	rm -f imports/imports.go
	rm -f cmd/platform/cprofile*.out
	rm -f cmd/mattermost/cprofile*.out

nuke: clean clean-docker ## Clean plus removes persistent server data.
	@echo BOOM

	rm -rf data

setup-mac: ## Adds macOS hosts entries for Docker.
	echo $$(boot2docker ip 2> /dev/null) dockerhost | sudo tee -a /etc/hosts

update-dependencies: ## Uses go get -u to update all the dependencies while holding back any that require it.
	@echo Updating Dependencies

	# Update all dependencies (does not update across major versions)
	$(GO) get -u

	# Tidy up
	$(GO) mod tidy

	# Copy everything to vendor directory
	$(GO) mod vendor


todo: ## Display TODO and FIXME items in the source code.
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime TODO
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime XXX
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime FIXME
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime "FIX ME"
ifeq ($(BUILD_ENTERPRISE_READY),true)
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime TODO enterprise/
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime XXX enterprise/
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime FIXME enterprise/
	@! ag --ignore Makefile --ignore-dir vendor --ignore-dir runtime "FIX ME" enterprise/
endif

## Help documentatin à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' ./Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
