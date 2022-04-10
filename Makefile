COMMIT_ID=$(shell git rev-parse --short HEAD)
VERSION=$(shell git tag --points-at HEAD)

ifeq ($(VERSION),)
	VERSION := $(COMMIT_ID)
endif

NAME=Noty
ICON_SET_PATH="./bin/$(NAME).app/Contents/icon.iconset"
ICON_OUTPUT_PATH="./bin/$(NAME).app/Contents/Resources/icon.png.png"
ICON_INPUT_PATH="./Resources/icon.png"

all: clean build

clean:
	@echo ">> cleaning..."
	@rm -rf ./bin/$(NAME).app

build: clean
	@echo ">> make app struct..."
	@mkdir "./bin/$(NAME).app"
	@mkdir "./bin/$(NAME).app/Contents"
	@mkdir "./bin/$(NAME).app/Contents/"{MacOS,Resources}
	@cp "./Resources/Info.plist" "./bin/$(NAME).app/Contents"
	@cp "./.env" "./bin/$(NAME).app/Contents/Resources/"

	@echo ">> make icon..."
	@mkdir $(ICON_SET_PATH)
	@qlmanage -z -t -s 1024 -o "./bin/$(NAME).app/Contents/Resources/" "$(ICON_INPUT_PATH)"
	@sips -z 16 16 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_16x16.png
	@sips -z 32 32 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_16x16@2x.png
	@cp $(ICON_SET_PATH)/icon_16x16@2x.png $(ICON_SET_PATH)/icon_32x32.png
	@sips -z 64 64 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_32x32@2x.png
	@sips -z 128 128 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_128x128.png
	@sips -z 256 256 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_128x128@2x.png
	@cp $(ICON_SET_PATH)/icon_128x128@2x.png $(ICON_SET_PATH)/icon_256x256.png
	@sips -z 512 512 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_256x256@2x.png
	@cp $(ICON_SET_PATH)/icon_256x256@2x.png $(ICON_SET_PATH)/icon_512x512.png
	@sips -z 1024 1024 "$(ICON_OUTPUT_PATH)" --out $(ICON_SET_PATH)/icon_512x512@2x.png
	@iconutil -c icns -o "./bin/$(NAME).app/Contents/Resources/icon.icns" $(ICON_SET_PATH)
	@rm -rf $(ICON_SET_PATH)
	@rm $(ICON_OUTPUT_PATH)

	@echo ">> building..."
	@env CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/$(NAME).app/Contents/MacOS/$(NAME) ./cmd/...
	@chmod +x ./bin/$(NAME).app/Contents/MacOS/$(NAME)

	@echo "Version: $(VERSION)"

install:
	@go install -ldflags "-X main.Version=$(VERSION) -X main.CommitID=$(COMMIT_ID)" ./cmd/...

.PHONY: all clean build install
