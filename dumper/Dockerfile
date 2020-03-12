FROM ubuntu:xenial

LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"

RUN apt update && apt install apt-transport-https -y

# install mysql client
RUN apt install -y mysql-client

# install mongo client
RUN apt install -y gnupg wget \
      && wget -qO - https://www.mongodb.org/static/pgp/server-4.2.asc | apt-key add - \
      && echo "deb [ arch=amd64 ] https://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/4.2 multiverse" | tee /etc/apt/sources.list.d/mongodb-org-4.2.list \
      && apt update \
      && apt install -y mongodb-org-shell mongodb-org-tools

COPY dumper .
ENTRYPOINT ["bash", "dumper"]
