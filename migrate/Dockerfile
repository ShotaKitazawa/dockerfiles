## Download dockerize stage ##
FROM alpine:3.10.3 as dockerize
LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"
ENV DOCKERIZE_VERSION v0.6.1
WORKDIR /workdir
# Download github.com/jwilder/dockerize
RUN apk add --no-cache openssl
RUN wget https://github.com/jwilder/dockerize/releases/download/$DOCKERIZE_VERSION/dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && tar -C /usr/local/bin -xzvf dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz \
    && rm dockerize-alpine-linux-amd64-$DOCKERIZE_VERSION.tar.gz

# migrate/migrate stage ##
FROM migrate/migrate:v4.11.0
# Copy dockerize binary
COPY --from=dockerize /usr/local/bin/dockerize /usr/local/bin/dockerize
ENTRYPOINT ["migrate"]

