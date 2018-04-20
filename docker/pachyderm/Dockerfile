FROM ubuntu:16.04

RUN apt-get update && apt-get install -y curl

RUN curl -o /tmp/pachctl.deb -L https://github.com/pachyderm/pachyderm/releases/download/v1.7.1/pachctl_1.7.1_amd64.deb && dpkg -i /tmp/pachctl.deb

ENTRYPOINT ["pachctl"]
