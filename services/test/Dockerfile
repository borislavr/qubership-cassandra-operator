FROM ghcr.io/netcracker/qubership-docker-integration-tests-debian:0.4.1

ENV ROBOT_OUTPUT=${ROBOT_HOME}/output \
    DISTR_DIR=/tmp/deps

RUN mkdir -p ${ROBOT_HOME} \
    && mkdir -p ${ROBOT_OUTPUT}

COPY requirements.txt ${ROBOT_HOME}/requirements.txt
COPY robot ${ROBOT_HOME}

RUN set -x \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        gcc \
        python3-dev \
        libffi-dev \
        passwd \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 999 cassandra && \
    useradd -u 999 -g cassandra -m -s /bin/bash cassandra

RUN set -x \
    && pip3 install --no-cache-dir -r ${ROBOT_HOME}/requirements.txt

RUN chown -R 999:999 ${ROBOT_HOME} \
    && chmod -R 775 ${ROBOT_HOME}

USER 999

EXPOSE 8080
VOLUME ["${ROBOT_HOME}"]