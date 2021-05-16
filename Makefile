APP_NAME := covaccine-notifier

BANNER:=\
    "\n"\
		"/**\n"\
    " * @project       $(APP_NAME)\n"\
    " */\n"\
    "\n"

## build.linux			: Build application for Linux runtime
.PHONY: build.linux
build.linux:
	env GOOS=linux go build -ldflags="-s -w" -o $(APP_NAME) .

## build.mac			: Build application for Mac runtime
.PHONY: build.mac
build.mac:
	env GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(APP_NAME) .

## help				: Show all available make targets
.PHONY : help
help : 
	@echo $(BANNER)
	@echo \ Make targets
	@echo -----------------------------
	@sed -n 's/^##//p' Makefile | sort
