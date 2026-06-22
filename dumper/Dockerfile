FROM ubuntu:24.04

LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates gnupg \
    mysql-client \
    etcd-client \
    && curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc \
       | gpg --dearmor -o /usr/share/keyrings/mongodb-server-8.0.gpg \
    && echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/ubuntu noble/mongodb-org/8.0 multiverse" \
       > /etc/apt/sources.list.d/mongodb-org-8.0.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-mongosh mongodb-database-tools \
    && rm -rf /var/lib/apt/lists/*

COPY dumper .
ENTRYPOINT ["bash", "dumper"]
