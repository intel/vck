FROM ubuntu:16.04

ENV KUBERNETES_SERVICE_HOST localhost
ENV KUBERNETES_SERVICE_PORT 443

COPY ./kube-volume-controller /
CMD /kube-volume-controller
