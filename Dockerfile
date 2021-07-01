FROM golang:1.16 as build

COPY . /app

RUN /app/build.sh

FROM alpine:latest
COPY --from=build /app/build /dist
