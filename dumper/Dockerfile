FROM ubuntu:24.04

LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"

RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-transport-https \
    && rm -rf /var/lib/apt/lists/*

# install mysql client
RUN apt-get update && apt-get install -y --no-install-recommends \
    mysql-client \
    && rm -rf /var/lib/apt/lists/*

# install mongo client
RUN apt-get update && apt-get install -y --no-install-recommends gnupg curl \
    && curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | gpg --dearmor -o /usr/share/keyrings/mongodb-server-7.0.gpg \
    && echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu noble/mongodb-org/7.0 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-7.0.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends mongodb-mongosh mongodb-database-tools \
    && rm -rf /var/lib/apt/lists/*

# install etcd client
RUN apt-get update && apt-get install -y --no-install-recommends \
    etcd-client \
    && rm -rf /var/lib/apt/lists/*

COPY dumper .
ENTRYPOINT ["bash", "dumper"]
