FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/qliksense-operator \
    USER_UID=1001 \
    USER_NAME=qliksense-operator

# install operator binary
COPY build/_output/bin/qliksense-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ARG KUBECTL_VERSION=1.17.0
#Install kubectl
RUN curl -LOv "http://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
RUN chmod +x kubectl
RUN mv kubectl /usr/local/bin/


ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
