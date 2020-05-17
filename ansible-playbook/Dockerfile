FROM centos:centos7.8.2003
LABEL maintainer="ShotaKitazawa <skitazawa1121@gmail.com>"
WORKDIR /workdir

RUN yum install -y python3 && \
    pip3 install ansible==2.9.7
RUN yum install -y openssh-clients rsync

COPY entrypoint.sh /usr/local/bin/
ENTRYPOINT ["entrypoint.sh"]
