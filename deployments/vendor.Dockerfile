FROM golang:1.17.2-buster as builder
RUN set -xe
RUN apt-get update
RUN apt-get install make gcc g++

WORKDIR /go/src/oms2
COPY go.mod go.sum /go/src/oms2/

RUN go mod download
