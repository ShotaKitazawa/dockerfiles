# renovate: datasource=docker depName=hashicorp/terraform
FROM hashicorp/terraform:1.12.2
RUN apk add --no-cache bash git openssh-client python3 py3-pip \
    && pip3 install --break-system-packages ansible-core
WORKDIR /workspace
