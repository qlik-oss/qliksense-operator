FROM golang:1.13-alpine
#RUN apt-get update
#RUN apt-get install gcc curl make -y
RUN mkdir -p /go/src/qlik-oss/qliksense-operator
COPY . /go/src/qlik-oss/qliksense-operator/
RUN go version
#RUN cd /go/src/qlik-oss/qliksense-operator && go test -v ./...
RUN cd /go/src/qlik-oss/qliksense-operator && go install

RUN cp /go/bin/qliksense-operator /usr/local/bin/
#FROM debian:stretch

#COPY --from=build /go/bin/qliksense-operator /usr/local/bin/
