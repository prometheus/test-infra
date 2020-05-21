FROM prominfra/infra:master
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

WORKDIR /funcbench
ENV INFRA_CMD infra

# Copy Makefiles and manifests
# Need 'cd' since ghActions ignores WORKDIR
# Need 'eval' to prevent bash keywords be run as commands
COPY ./funcbench /bin/funcbench
COPY ./ ./
RUN echo -e '#!/bin/sh\ncd /funcbench\neval "$@"' >/bin/docker_entrypoint
RUN chmod u+x /bin/docker_entrypoint

ENTRYPOINT ["/bin/funcbench"]
