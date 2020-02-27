FROM golang:1.13-stretch as build
RUN apt-get update
RUN apt-get install gcc curl make -y
RUN mkdir -p /go/src/qlik-oss/qliksense-operator
COPY . /go/src/qlik-oss/qliksense-operator/
RUN go version
RUN cd /go/src/qlik-oss/qliksense-operator && go test -v ./...
RUN cd /go/src/qlik-oss/qliksense-operator && go install

FROM debian:stretch

ARG KUBECTL_VERSION=1.17.0
#Install kubectl
RUN curl -LOv "https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
RUN chmod +x kubectl
RUN mv kubectl /usr/local/bin/

COPY --from=build /go/bin/qliksense-operator /usr/local/bin/
