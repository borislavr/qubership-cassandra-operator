FROM thelastpickle/cassandra-reaper:4.0.1

ENV USER_ID=999

USER root
ADD run.sh /

RUN echo 'https://dl-cdn.alpinelinux.org/alpine/edge/main/' > /etc/apk/repositories \
    && apk add --no-cache openssl \
    && apk update \
    && apk upgrade

RUN chown ${USER_ID}:0 \
    /etc/cassandra-reaper/cassandra-reaper.yml \
    /usr/local/lib/cassandra-reaper.jar \
    /usr/local/bin/entrypoint.sh \
    /usr/local/bin/configure-persistence.sh \
    /usr/local/bin/configure-metrics.sh \
    /usr/local/bin/configure-jmx-credentials.sh && \
    chown -R ${USER_ID}:0 \
    /var/lib/cassandra-reaper \
    /usr/local/bin/entrypoint.sh \
    /etc/cassandra-reaper/shiro \
    /var/log/cassandra-reaper

RUN chmod +x /run.sh
RUN chown ${USER_ID}:0 /run.sh

RUN chmod 777 /etc/cassandra-reaper/config

VOLUME /etc/cassandra-reaper/config

USER ${USER_ID}

ENTRYPOINT ["/run.sh"]