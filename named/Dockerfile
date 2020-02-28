FROM alpine:3.10.3

RUN apk add bash --no-cache

# install bind
RUN apk add bind bind-tools --no-cache
RUN echo 'nameserver 127.0.0.1' > /etc/resolv.conf
ADD named/nsupdate.sh /

CMD ["/usr/sbin/named", "-4", "-c", "/etc/named/named.conf", "-u", "root", "-g"]
