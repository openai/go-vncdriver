transcode:
	go build --ldflags '-extldflags "-static"'

image: transcode
	docker build -t docker.openai.com/transcode .

push:
	docker push docker.openai.com/transcode


.PHONY: transcode image push
