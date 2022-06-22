ARG BASE_IMG=ubuntu:22.04

FROM $BASE_IMG
    WORKDIR /app

    RUN apt-get update && \
        apt-get install -y \
            ca-certificates && \
        apt-get autoremove -y && \
        apt-get clean -y && \
        rm -rf /var/cache/apt/archives /var/lib/apt/lists/*
        
    COPY out/api api

    CMD ./api
