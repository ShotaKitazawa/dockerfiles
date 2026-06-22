FROM ubuntu:24.04
LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"

RUN apt-get update && apt-get install -y --no-install-recommends \
    tcpdump iputils-ping net-tools dnsutils telnet curl \
    && rm -rf /var/lib/apt/lists/*

CMD ["sleep", "86400"]
