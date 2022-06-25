# The base image used to build all other images
ARG BASE_IMG=ubuntu:22.04
# The tag to use for golang image
ARG GOLANG_TAG=1.18.3

#
# Download and install all deps required to run tests and build the go application
#
FROM golang:$GOLANG_TAG as go

FROM $BASE_IMG as go-builder
    WORKDIR /tmp/build

    # update the apt repo and install any deps we might need.
    RUN apt-get update && \
        apt-get install -y \
            build-essential \
            make \
            git && \
        apt-get autoremove -y && \
        apt-get clean -y && \
        rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

    ENV PATH /usr/local/go/bin:$PATH
    ENV GOPATH /go
    ENV PATH $GOPATH/bin:$PATH
    COPY --from=go /usr/local /usr/local
    COPY --from=go /go /go

    COPY go.mod .
    COPY go.sum .
    COPY Makefile .

    RUN mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH" && \
        make deps

    COPY . .

    RUN make generate && \
        make test

    ARG BUILDER
    ARG VERSION

    ENV IMAGES_BUILDER=${BUILDER}
    ENV IMAGES_VERSION=${VERSION}

    RUN make

#
# final squashed image
#
FROM $BASE_IMG as final
    WORKDIR /app

    RUN apt-get update && \
        apt-get install -y \
            ca-certificates && \
        apt-get autoremove -y && \
        apt-get clean -y && \
        rm -rf /var/cache/apt/archives /var/lib/apt/lists/*

    COPY --from=go-builder /tmp/build/out .

    CMD ./api
