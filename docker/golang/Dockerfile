FROM golang:1.9.2

ENV DEBIAN_FRONTEND=noninteractive

# Set timezone to UTC by default
RUN ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime

# Need socat to forward user ssh-agent
RUN apt-get update && apt-get upgrade -y && \
    apt-get install -y --no-install-recommends git socat && \
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
