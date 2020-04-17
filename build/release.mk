
dist: | check-style test package

build-linux:
	@echo Build Linux amd64
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(GOBIN) $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' ./...
else
	mkdir -p $(GOBIN)/linux_amd64
	env GOOS=linux GOARCH=amd64 $(GO) build -o $(GOBIN)/linux_amd64 $(GOFLAGS) -trimpath -ldflags '$(LDFLAGS)' ./...
endif

build: build-linux

build-client:
	@echo Building mattermost web app

	cd $(BUILD_WEBAPP_DIR) && $(MAKE) build

package:
	@ echo Packaging mattermost

	@# Remove any old files
	rm -Rf $(DIST_ROOT)

	@# Create needed directories
	mkdir -p $(DIST_PATH)/bin
	mkdir -p $(DIST_PATH)/logs
	mkdir -p $(DIST_PATH)/prepackaged_plugins

	@# Resource directories
	mkdir -p $(DIST_PATH)/config
	cp -L config/README.md $(DIST_PATH)/config
	OUTPUT_CONFIG=$(PWD)/$(DIST_PATH)/config/config.json go generate ./config
	cp -RL fonts $(DIST_PATH)
	cp -RL templates $(DIST_PATH)
	cp -RL i18n $(DIST_PATH)

	@# Disable developer settings
	sed -i'' -e 's|"ConsoleLevel": "DEBUG"|"ConsoleLevel": "INFO"|g' $(DIST_PATH)/config/config.json
	sed -i'' -e 's|"SiteURL": "http://localhost:8065"|"SiteURL": ""|g' $(DIST_PATH)/config/config.json

	@# Reset email sending to original configuration
	sed -i'' -e 's|"SendEmailNotifications": true,|"SendEmailNotifications": false,|g' $(DIST_PATH)/config/config.json
	sed -i'' -e 's|"FeedbackEmail": "test@example.com",|"FeedbackEmail": "",|g' $(DIST_PATH)/config/config.json
	sed -i'' -e 's|"ReplyToAddress": "test@example.com",|"ReplyToAddress": "",|g' $(DIST_PATH)/config/config.json
	sed -i'' -e 's|"SMTPServer": "localhost",|"SMTPServer": "",|g' $(DIST_PATH)/config/config.json
	sed -i'' -e 's|"SMTPPort": "2500",|"SMTPPort": "",|g' $(DIST_PATH)/config/config.json

	@# Package webapp
	mkdir -p $(DIST_PATH)/client
	cp -RL $(BUILD_WEBAPP_DIR)/dist/* $(DIST_PATH)/client

	@# Help files
	cp build/MIT-COMPILED-LICENSE.md $(DIST_PATH)
	cp NOTICE.txt $(DIST_PATH)
	cp README.md $(DIST_PATH)

	@# Download prepackaged plugins
	mkdir -p tmpprepackaged
	@cd tmpprepackaged && for plugin_package in $(PLUGIN_PACKAGES) ; do \
		curl -O -L https://plugins-store.test.mattermost.com/release/$$plugin_package.tar.gz; \
		curl -O -L https://plugins-store.test.mattermost.com/release/$$plugin_package.tar.gz.sig; \
	done

	@# ----- PLATFORM SPECIFIC -----

	@# Make linux package
	@# Copy binary
ifeq ($(BUILDER_GOOS_GOARCH),"linux_amd64")
	cp $(GOBIN)/mattermost $(DIST_PATH)/bin # from native bin dir, not cross-compiled
	cp $(GOBIN)/platform $(DIST_PATH)/bin # from native bin dir, not cross-compiled
else
	cp $(GOBIN)/linux_amd64/mattermost $(DIST_PATH)/bin # from cross-compiled bin dir
	cp $(GOBIN)/linux_amd64/platform $(DIST_PATH)/bin # from cross-compiled bin dir
endif
	@# Strip and prepackage plugins
	@for plugin_package in $(PLUGIN_PACKAGES) ; do \
		cat tmpprepackaged/$$plugin_package.tar.gz | gunzip | tar --wildcards --delete "*windows*" --delete "*darwin*" | gzip  > $(DIST_PATH)/prepackaged_plugins/$$plugin_package.tar.gz; \
		cp tmpprepackaged/$$plugin_package.tar.gz.sig $(DIST_PATH)/prepackaged_plugins; \
	done
	@# Package
	tar -C dist -czf $(DIST_PATH)-$(BUILD_TYPE_NAME)-linux-amd64.tar.gz mattermost
	@# Don't clean up native package so dev machines will have an unzipped package available
	@#rm -f $(DIST_PATH)/bin/mattermost

	rm -rf tmpprepackaged
