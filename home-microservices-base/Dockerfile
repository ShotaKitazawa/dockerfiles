# base image
FROM ubuntu:20.04
LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"
WORKDIR /opt
## install from apt
RUN apt update -y && \
      apt install curl -y
## install jq
RUN JQ_VERSION=1.6 && \
      curl -sL -o /usr/local/bin/jq https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 && \
      chmod +x /usr/local/bin/jq
## install yq
RUN YQ_VERSION=v4.2.1 && \
      curl -sL -o /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_linux_amd64 && \
      chmod +x /usr/local/bin/yq
## install grpcurl
RUN GRPCURL_VERSION=1.6.0 && \
      curl -sL https://github.com/fullstorydev/grpcurl/releases/download/v${GRPCURL_VERSION}/grpcurl_${GRPCURL_VERSION}_linux_x86_64.tar.gz | tar xzv -C /opt && \
      mv /opt/grpcurl /usr/local/bin/
## install grpc_health_probe
RUN GRPC_HEALTH_PROBE_VERSION=v0.3.1 && \
    curl -sL -o /usr/local/bin/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-amd64 && \
    chmod +x /usr/local/bin/grpc_health_probe
USER nobody
