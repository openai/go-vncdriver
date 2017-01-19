FROM golang
RUN apt-get update \
    && apt-get install -y libjpeg62-turbo-dev \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
COPY . /go/src/github.com/openai/go-vncdriver
RUN go install --ldflags '-extldflags "-static"' github.com/openai/go-vncdriver/cmd/transcode
