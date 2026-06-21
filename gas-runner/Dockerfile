# renovate: datasource=docker depName=node
FROM node:20-alpine
RUN apk upgrade --no-cache \
    && apk add --no-cache bash git openssh-client python3 py3-pip py3-yaml \
    && pip3 install --break-system-packages ansible-core \
    && npm install -g @google/clasp
WORKDIR /workspace
