FROM ubuntu:20.04

LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"

RUN apt update && apt install apt-transport-https -y

# install mysql client
RUN apt install -y mysql-client

# install mongo client
RUN apt install -y gnupg wget \
      && wget -qO - https://www.mongodb.org/static/pgp/server-4.4.asc | apt-key add - \
      && echo "deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.4.list \
      && apt update \
      && apt install -y mongodb-org-shell mongodb-org-tools

# install etcd client
RUN apt install -y etcd-client

COPY dumper/dumper .
ENTRYPOINT ["bash", "dumper"]
