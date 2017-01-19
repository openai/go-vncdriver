URL ?= docker.openai.com/transcode

transcode:
	go build

build: transcode

image:
	cd ../.. && docker build --tag transcode-builder -f cmd/transcode/builder.dockerfile .
	docker run --rm transcode-builder cat /go/bin/transcode > transcode
	chmod +x transcode
	docker build -t $(URL):latest .
	docker tag $(URL):latest $(URL):$$(cat VERSION)

push:
	docker push $(URL):latest
	docker push $(URL):$$(cat VERSION)


.PHONY: transcode image push
