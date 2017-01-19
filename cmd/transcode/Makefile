URL ?= docker.openai.com/transcode

transcode:
	go build --ldflags '-extldflags "-static"'

image: transcode
	docker build -t $(URL):latest .
	docker tag $(URL):latest $(URL):$$(cat VERSION)

push:
	docker push $(URL):latest
	docker push $(URL):$$(cat VERSION)


.PHONY: transcode image push
