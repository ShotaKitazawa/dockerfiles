# renovate: datasource=docker depName=hashicorp/terraform
FROM hashicorp/terraform:1.16.1
# renovate: datasource=pypi depName=ansible-core
ARG ANSIBLE_CORE_VERSION=2.21.1
RUN apk upgrade --no-cache libexpat \
    && apk add --no-cache bash git openssh-client python3 py3-pip \
    && pip3 install --break-system-packages --no-cache-dir ansible-core==${ANSIBLE_CORE_VERSION}
WORKDIR /workspace
