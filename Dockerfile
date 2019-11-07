FROM golang:stretch as build
ENV GO111MODULE=on
RUN apt-get update
RUN apt-get install gcc curl make -y 
RUN mkdir -p /go/src/qlik-oss/qliksense-operator
COPY . /go/src/qlik-oss/qliksense-operator/
RUN cd /go/src/qlik-oss/qliksense-operator && go install

FROM debian:stretch

COPY --from=build /go/bin/qliksense-operator /usr/local/bin/
