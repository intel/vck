FROM ubuntu:16.04

ENV KUBERNETES_SERVICE_HOST localhost
ENV KUBERNETES_SERVICE_PORT 443

RUN mkdir -p /kvc-templates
COPY ./pkg/resources /kvc-templates
COPY ./kube-volume-controller /
CMD /kube-volume-controller
