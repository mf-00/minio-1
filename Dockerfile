FROM golang:1.7

WORKDIR /go/src/app

COPY . /go/src/app
RUN \
	go-wrapper download && \
	go-wrapper install -ldflags "$(go run buildscripts/gen-ldflags.go)" && \
	mkdir -p /export/docker && \
	cp /go/src/app/docs/Docker.md /export/docker/ && \

EXPOSE 9000
ENV MINIO_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
ENV MINIO_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
ENTRYPOINT ["go-wrapper", "run", "server"]
VOLUME ["/export"]
