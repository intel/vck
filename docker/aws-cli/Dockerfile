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

FROM alpine:3.7
ENV AWS_ENDPOINT_URL=""
RUN apk --update add python py-pip \
      groff less mailcap && \
    pip install --upgrade awscli==1.14.37 s3cmd==2.0.1 python-magic && \
    apk --purge del py-pip && rm /var/cache/apk/*
VOLUME /root/.aws
COPY aws_wrapper /usr/local/bin/aws
ENTRYPOINT ["aws"]
