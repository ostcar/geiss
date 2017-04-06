BUILD = go build -ldflags "-w -s -X main.Version=$(VERSION)"
GITHUB_USER = ostcar
GITHUB_REPO = geiss
UPLOAD_FILE = github-release upload \
	--user $(GITHUB_USER) \
	--repo $(GITHUB_REPO) \
	--tag $(VERSION)

build:
	$(eval VERSION := $(shell git describe --tags))
	$(BUILD)

release: clean set_version build_release
	github-release release \
		--user $(GITHUB_USER) \
		--repo $(GITHUB_REPO) \
		--tag $(VERSION)
	$(UPLOAD_FILE) --name geiss_linux_32 --file release/geiss_linux_32
	$(UPLOAD_FILE) --name geiss_linux_64 --file release/geiss_linux_64
	$(UPLOAD_FILE) --name geiss_mac_32 --file release/geiss_mac_32
	$(UPLOAD_FILE) --name geiss_mac_64 --file release/geiss_mac_64
	$(UPLOAD_FILE) --name geiss_windows_32.exe --file release/geiss_windows_32.exe
	$(UPLOAD_FILE) --name geiss_windows_64.exe --file release/geiss_windows_64.exe

set_version:
ifndef VERSION
	$(error The environment varialbe VERSION is not defined)
endif
	$(info Crate git tag ${VERSION})
	git tag -s ${VERSION}
	git push --tags

clean:
	rm -fr release

build_release: release/geiss_linux_32 release/geiss_linux_64 release/geiss_windows_32.exe release/geiss_windows_64.exe release/geiss_mac_32 release/geiss_mac_64

release/geiss_linux_32:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=linux GOARCH=386 $(BUILD) -o $@

release/geiss_linux_64:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=linux GOARCH=amd64 $(BUILD) -o $@

release/geiss_windows_32.exe:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=windows GOARCH=386 $(BUILD) -o $@

release/geiss_windows_64.exe:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=windows GOARCH=amd64 $(BUILD) -o $@

release/geiss_mac_32:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=darwin GOARCH=386 $(BUILD) -o $@

release/geiss_mac_64:
	$(eval VERSION := $(shell git describe --tags))
	GOOS=darwin GOARCH=amd64 $(BUILD) -o $@
