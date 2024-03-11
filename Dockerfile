# syntax=docker/dockerfile:1

FROM golang:1.21 AS build

WORKDIR $GOPATH/src/github.com/brotherlogic/addrecord

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 go build -o /addrecord

##
## Deploy
##
FROM gcr.io/distroless/base-debian11

WORKDIR /

COPY --from=build /addrecord /addrecord

USER nonroot:nonroot

ENTRYPOINT ["/addrecord"]