#
# Copyright (c) 2018 Intel Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# SPDX-License-Identifier: EPL-2.0
#

FROM golang:1.9.2

ARG GCLOUD_SDK_VERSION="192.0.0"
ARG HELM_VERSION="v2.6.1"

ENV DEBIAN_FRONTEND=noninteractive

# Set timezone to UTC by default
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime

# Need socat to forward user ssh-agent
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y --no-install-recommends git socat jq && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN mkdir -p /root/.ssh && \
    touch /root/.ssh/known_hosts && \
    ssh-keyscan -t rsa github.com >> /root/.ssh/known_hosts

RUN go get -u \
    github.com/golang/dep/cmd/dep \
    github.com/alecthomas/gometalinter \
    github.com/kubernetes/gengo/examples/deepcopy-gen \
    && gometalinter --install

# NOTE: Install gcloud sdk
RUN curl -sSLo google-cloud-sdk.tar.gz https://storage.googleapis.com/cloud-sdk-release/google-cloud-sdk-${GCLOUD_SDK_VERSION}-linux-x86_64.tar.gz &&\
    tar zxf google-cloud-sdk.tar.gz && rm google-cloud-sdk.tar.gz &&\
    ./google-cloud-sdk/install.sh --usage-reporting=true --path-update=true --quiet && \
    /go/google-cloud-sdk/bin/gcloud components install kubectl &&\
    ln -s "/go/google-cloud-sdk/bin/gcloud" /usr/local/bin/ &&\
    ln -s "/go/google-cloud-sdk/bin/kubectl" /usr/local/bin/

# NOTE: Install helm
RUN curl -sfL http://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz -o /tmp/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar xzf /tmp/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin && \
    rm /tmp/helm-${HELM_VERSION}-linux-amd64.tar.gz && rm -r linux-amd64

# NOTE: Install docker/docker-compose same as circleci
RUN set -ex \
  && export DOCKER_VERSION=$(curl --silent --fail --retry 3 https://download.docker.com/linux/static/stable/x86_64/ | grep -o -e 'docker-[.0-9]*-ce\.tgz' | sort -r | head -n 1) \
  && DOCKER_URL="https://download.docker.com/linux/static/stable/x86_64/${DOCKER_VERSION}" \
  && echo Docker URL: $DOCKER_URL \
  && curl --silent --show-error --location --fail --retry 3 --output /tmp/docker.tgz "${DOCKER_URL}" \
  && ls -lha /tmp/docker.tgz \
  && tar -xz -C /tmp -f /tmp/docker.tgz \
  && mv /tmp/docker/* /usr/bin \
  && rm -rf /tmp/docker /tmp/docker.tgz \
  && which docker \
  && (docker version || true)
RUN COMPOSE_URL="https://circle-downloads.s3.amazonaws.com/circleci-images/cache/linux-amd64/docker-compose-latest" \
  && curl --silent --show-error --location --fail --retry 3 --output /usr/bin/docker-compose $COMPOSE_URL \
  && chmod +x /usr/bin/docker-compose \
  && docker-compose version
