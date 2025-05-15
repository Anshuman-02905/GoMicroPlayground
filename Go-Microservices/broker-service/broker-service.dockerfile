# base go image
FROM golang:1.18-alpine as builder


# build a tiny docker image
FROM alpine:latest

RUN mkdir /app

COPY brokerApp /app

CMD [ "/app/brokerApp" ]