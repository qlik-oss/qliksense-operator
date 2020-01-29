FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/qliksense-operator \
    USER_UID=1001 \
    USER_NAME=qliksense-operator

# install operator binary
COPY build/_output/bin/qliksense-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
