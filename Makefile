OPEND_URL ?= https://softwaredownload.futunn.com/Futu_OpenD_10.2.6208_Ubuntu18.04.tar.gz
OPEND_DIR ?= .opend
OPEND_DOWNLOAD_DIR := $(OPEND_DIR)/downloads
OPEND_EXTRACT_DIR := $(OPEND_DIR)/app
OPEND_ARCHIVE := $(OPEND_DOWNLOAD_DIR)/$(notdir $(OPEND_URL))
OPEND_SETUP_STAMP := $(OPEND_EXTRACT_DIR)/.setup-complete

.PHONY: gen clean opend-download opend-setup opend-clean

gen:
	@command -v buf >/dev/null 2>&1 || { echo "Buf is not installed. Install it with 'brew install buf'."; exit 1; }
	buf generate

clean:
	@echo "Cleaning generated files..."
	rm -rf gen

opend-download: $(OPEND_ARCHIVE)
	@echo "OpenD archive ready at $(OPEND_ARCHIVE)"

$(OPEND_ARCHIVE):
	@mkdir -p $(OPEND_DOWNLOAD_DIR)
	curl --fail --location --retry 3 --output $@ $(OPEND_URL)

opend-setup: $(OPEND_SETUP_STAMP)
	@echo "OpenD extracted to $(OPEND_EXTRACT_DIR)"

$(OPEND_SETUP_STAMP): $(OPEND_ARCHIVE)
	@rm -rf $(OPEND_EXTRACT_DIR)
	@mkdir -p $(OPEND_EXTRACT_DIR)
	tar -xzf $(OPEND_ARCHIVE) -C $(OPEND_EXTRACT_DIR)
	@touch $@

opend-clean:
	@echo "Removing local OpenD files..."
	rm -rf $(OPEND_DIR)
