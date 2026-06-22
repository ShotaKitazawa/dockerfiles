# renovate: datasource=docker depName=hashicorp/terraform
FROM hashicorp/terraform:1.12.2
RUN apk upgrade --no-cache libexpat \
    && apk add --no-cache bash git openssh-client python3 py3-pip \
    # renovate: datasource=pypi depName=ansible-core
    && pip3 install --break-system-packages --no-cache-dir ansible-core==2.21.1
WORKDIR /workspace
