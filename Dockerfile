FROM golang:1.7

WORKDIR /go/src/app
ENV ALLOW_CONTAINER_ROOT=1

COPY . /go/src/app
RUN \
	go-wrapper download && \
	go-wrapper install && \
	mkdir -p /export/docker

EXPOSE 9000
ENV MINIO_ACCESS_KEY="AKIAIOSFODNN7EXAMPLE"
ENV MINIO_SECRET_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
ENTRYPOINT ["go-wrapper", "run", "server"]
VOLUME ["/export"]
CMD ["/export"]
