FROM alpine:3.10.3

RUN apk add bind --no-cache
RUN echo 'nameserver 127.0.0.1' > /etc/resolv.conf
ADD named/nsupdate.sh /

CMD ["/usr/sbin/named", "-c", "/etc/named/named.conf", "-u", "named", "-g"]
