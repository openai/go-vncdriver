URL ?= docker.openai.com/transcode

transcode:
	go build

build: transcode

image:
	@if [ $(shell uname) = Darwin ]; then \
		echo "-----------------------------------"; \
		echo "THIS IMAGE MUST BE BUILT ON LINUX!!"; \
		echo "-----------------------------------"; \
		exit 1; \
	fi
	go build --ldflags '-extldflags "-static"'
	docker build -t $(URL):latest .
	docker tag $(URL):latest $(URL):$$(cat VERSION)

push:
	docker push $(URL):latest
	docker push $(URL):$$(cat VERSION)


.PHONY: transcode image push
