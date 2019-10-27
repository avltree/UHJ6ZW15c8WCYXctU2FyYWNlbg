FROM golang:1.13-alpine

RUN apk update && apk add bash git

RUN mkdir /gwp-api
WORKDIR /gwp-api

CMD ["tail"]
