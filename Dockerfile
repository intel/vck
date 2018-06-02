FROM ubuntu:16.04

ENV KUBERNETES_SERVICE_HOST localhost
ENV KUBERNETES_SERVICE_PORT 443

RUN mkdir -p /vck-templates
COPY ./pkg/templates /vck-templates
COPY ./vck /
CMD /vck
