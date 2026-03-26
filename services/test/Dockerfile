FROM ghcr.io/netcracker/qubership-docker-integration-tests-debian:0.4.0

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
    && rm -rf /var/lib/apt/lists/*


RUN set -x \
    && pip3 install --no-cache-dir -r ${ROBOT_HOME}/requirements.txt

USER root
RUN chmod -R 777 ${ROBOT_HOME}

EXPOSE 8080
VOLUME ["${ROBOT_HOME}"]