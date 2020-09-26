FROM golang:latest

RUN go get -u -v -x github.com/ipsn/go-libtor && \
    go get -u github.com/cretz/bine/tor       && \
    go get -u github.com/armon/go-socks5

COPY . /offensive-tor-toolkit

WORKDIR /offensive-tor-toolkit
RUN /offensive-tor-toolkit/.build.sh

COPY ./.entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
