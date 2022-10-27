FROM docker.io/debian:bullseye-slim

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

ARG APP_VERSION

RUN apt-get update -qq \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update -qq \
    && apt-get install -y --no-install-recommends kubectl=${APP_VERSION}-00 \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists /var/cache/apt/archives

ENV HOME="/" 
RUN mkdir ${HOME}/.kube && chmod g+rwX ${HOME}.kube
USER 1001

ENTRYPOINT [ "kubectl" ]
CMD [ "--help" ]