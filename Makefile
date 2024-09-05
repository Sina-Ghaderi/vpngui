GO_PATH_DIR := $(shell go env GOPATH)
INNO_SETUP_ISCC_PATH := /c/Program\ Files\ \(x86\)/Inno\ Setup\ 6/iscc
GO_ENV := GO_ENABLED=0 GOOS=windows
BUILD_FLAGS := -ldflags="-w -s -buildid= -H windowsgui"
SNIXCONNECT_DIR := snixconnect
BIN_DIR := bin

GO1_20_DL := https://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-go-1.20.2-1-any.pkg.tar.zst
GO1_23_DL := https://repo.msys2.org/mingw/x86_64/mingw-w64-x86_64-go-1.23.0-1-any.pkg.tar.zst
RESOURCE_DIR := resource

BIN_SNIXCONNECT32 := snixconnect-x86.exe
BIN_SNIXCONNECT64 := snixconnect-x64.exe
BIN_MANAGER32 := snixmanager-x86.exe
BIN_MANAGER64 := snixmanager-x64.exe
BIN_SERVICE32 := snixservice-x86.exe
BIN_SERVICE64 := snixservice-x64.exe



.PHONY: all clean
all: win78 win10 installer

win78:
	mkdir -p $(RESOURCE_DIR) && wget -c $(GO1_20_DL) -O $(RESOURCE_DIR)/mingw-w64-x86_64-go-1.20.2-1-any.pkg.tar.zst
	pacman -U $(RESOURCE_DIR)/mingw-w64-x86_64-go-1.20.2-1-any.pkg.tar.zst --noconfirm
	cat mods/go.mod.old > go.mod
	cat mods/go.sum.old > go.sum
	$(MAKE) dependency
	$(MAKE) executable
	$(MAKE) manager
	$(MAKE) service
	cd $(BIN_DIR) && mv $(BIN_SNIXCONNECT32) snixconnect-old-x86.exe
	cd $(BIN_DIR) && mv $(BIN_SNIXCONNECT64) snixconnect-old-x64.exe
	cd $(BIN_DIR) && mv $(BIN_MANAGER32) snixmanager-old-x86.exe
	cd $(BIN_DIR) && mv $(BIN_MANAGER64) snixmanager-old-x64.exe
	cd $(BIN_DIR) && mv $(BIN_SERVICE32) snixservice-old-x86.exe
	cd $(BIN_DIR) && mv $(BIN_SERVICE64) snixservice-old-x64.exe

win10:
	mkdir -p $(RESOURCE_DIR) && wget -c $(GO1_23_DL) -O $(RESOURCE_DIR)/mingw-w64-x86_64-go-1.23.0-1-any.pkg.tar.zst
	pacman -U $(RESOURCE_DIR)/mingw-w64-x86_64-go-1.23.0-1-any.pkg.tar.zst --noconfirm
	cat mods/go.mod.current > go.mod
	cat mods/go.sum.current > go.sum
	$(MAKE) dependency
	$(MAKE) executable
	$(MAKE) manager
	$(MAKE) service

dependency:
	go mod tidy
	go install github.com/akavel/rsrc@latest

executable:
	cd $(SNIXCONNECT_DIR) && $(GO_PATH_DIR)/bin/rsrc -manifest app.xml -ico app.ico -o rsrc.syso
	cd $(SNIXCONNECT_DIR) && GOARCH=386 $(GO_ENV) go build $(BUILD_FLAGS) -trimpath -o ../$(BIN_DIR)/$(BIN_SNIXCONNECT32)
	cd $(SNIXCONNECT_DIR) && GOARCH=amd64 $(GO_ENV) go build $(BUILD_FLAGS) -trimpath -o ../$(BIN_DIR)/$(BIN_SNIXCONNECT64)


manager:
	cd $(SNIXCONNECT_DIR)/manager && $(GO_PATH_DIR)/bin/rsrc -manifest manager.xml -ico ../app.ico -o rsrc.syso
	cd $(SNIXCONNECT_DIR)/manager && GOARCH=386 $(GO_ENV) go build -tags manager $(BUILD_FLAGS) -trimpath -o ../../$(BIN_DIR)/$(BIN_MANAGER32)
	cd $(SNIXCONNECT_DIR)/manager && GOARCH=amd64 $(GO_ENV) go build -tags manager $(BUILD_FLAGS) -trimpath -o ../../$(BIN_DIR)/$(BIN_MANAGER64)


service:
	cd $(SNIXCONNECT_DIR)/service && GOARCH=386 $(GO_ENV) go build -tags service -ldflags="-w -s" -trimpath -o ../../$(BIN_DIR)/$(BIN_SERVICE32)
	cd $(SNIXCONNECT_DIR)/service && GOARCH=amd64 $(GO_ENV) go build -tags service -ldflags="-w -s" -trimpath -o ../../$(BIN_DIR)/$(BIN_SERVICE64)

installer:
	cd $(BIN_DIR) && $(INNO_SETUP_ISCC_PATH) installer.iss

clean:
	-cd $(SNIXCONNECT_DIR) && rm rsrc.syso
	-cd $(SNIXCONNECT_DIR)/manager && rm rsrc.syso
	-cd $(BIN_DIR) && rm *.exe
	-cd $(BIN_DIR) && rm -rf output
	-rm -rf $(RESOURCE_DIR)