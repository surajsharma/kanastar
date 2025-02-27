# Define variables
GOOS_LIST = linux darwin
GOARCH_LIST = amd64 arm64
BINARY_NAME = kanactl
VERSION ?= $(shell git describe --tags --always)
DIST_DIR = dist

# Default target
.PHONY: all build clean release

all: build

# Build binaries for all OS/ARCH combinations
build:
	@for GOOS in $(GOOS_LIST); do \
		for GOARCH in $(GOARCH_LIST); do \
			GOOS=$$GOOS GOARCH=$$GOARCH go build -o $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH; \
			chmod +x $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH; \
			tar -czvf $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-$$GOOS-$$GOARCH; \
			rm -f $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH; \
		done; \
	done

# Clean built files
clean:
	rm -rf $(DIST_DIR)

# Create a GitHub Release (requires GitHub CLI)
#release: build
#	gh release create $(VERSION) $(BINARY_NAME)-*.tar.gz --title "Release $(VERSION)" --notes "Automated release of $(VERSION)"
