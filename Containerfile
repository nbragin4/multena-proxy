FROM registry.access.redhat.com/ubi8/ubi-minimal:8.7-1085
WORKDIR /app/
ENV GOPS_CONFIG_DIR /app/.config
RUN mkdir /app/.config
COPY  multena-proxy .
RUN chgrp -R 0 /app && chmod -R g=u /app
EXPOSE 8080
ENTRYPOINT [ "/app/multena-proxy" ]