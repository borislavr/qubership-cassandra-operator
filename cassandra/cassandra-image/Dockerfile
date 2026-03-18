FROM eclipse-temurin:11-jre

ARG version
ENV CASSANDRA_VERSION $version

ARG exp_version
ENV EXPORTER_VERSION $exp_version

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        wget \
        net-tools \
        jq \
        openssh-server \
        bash \
        python3 \
        python3-dev \
        python3-pip \
        libev-dev \
        build-essential \
        libffi-dev \
        libssl-dev \
        rsync \
        libarchive-tools \
        grep \
        openssl \
    && rm -rf /var/lib/apt/lists/*

RUN sed -i "s/999/99/" /etc/group

RUN ln -s /opt/java/openjdk/bin/java /usr/bin/java

ENV CASSANDRA_CONFIG_DIR /opt/cassandra/conf
ENV CASSANDRA_INIT_CONFIG_DIR /var/lib/cassandra/configuration
ENV CASSANDRA_DATA /var/lib/cassandra/data
ENV CASSANDRA_HOME /opt/cassandra

COPY pip.conf /etc/pip.conf
RUN pip3 install cassandra-driver cqlsh --break-system-packages

RUN wget -qO- https://archive.apache.org/dist/cassandra/${CASSANDRA_VERSION}/apache-cassandra-${CASSANDRA_VERSION}-bin.tar.gz | tar xvfz - -C /tmp/ && mv /tmp/apache-cassandra-${CASSANDRA_VERSION} $CASSANDRA_HOME
ENV PATH $PATH:$CASSANDRA_HOME/bin:$CASSANDRA_HOME/tools/bin

RUN echo 'export PATH=$PATH:'"$CASSANDRA_HOME/bin:$CASSANDRA_HOME/tools/bin" > $CASSANDRA_HOME/.profile 
#RUN cp $CASSANDRA_HOME/bin/cqlsh /usr/share/bin

RUN mkdir -p /usr/share/java/

RUN wget -O /usr/share/java/sjk-plus-0.17.jar https://repo1.maven.org/maven2/org/gridkit/jvmtool/sjk-plus/0.17/sjk-plus-0.17.jar
RUN wget -O /usr/share/java/cassandra-exporter-agent.jar https://github.com/instaclustr/cassandra-exporter/releases/download/${EXPORTER_VERSION}
RUN wget -O $CASSANDRA_HOME/lib/netty-codec-http-4.1.96.Final.jar https://repo1.maven.org/maven2/io/netty/netty-codec-http/4.1.96.Final/netty-codec-http-4.1.96.Final.jar

RUN rm -f $CASSANDRA_CONFIG_DIR/cassandra-topology.properties

RUN mkdir -p /var/lib/cassandra \
    && mkdir -p /var/lib/cassandra/custom_ssh \
    && chmod -R 777 /var/lib/cassandra \
    && chmod -R 777 /var/lib/cassandra \
    && chmod -R 777 $CASSANDRA_CONFIG_DIR \
    && chmod -R 777 $CASSANDRA_HOME \
    && chmod 777 /etc/passwd \
    && chmod -R 777 /var/lib/cassandra/custom_ssh

VOLUME /var/lib/cassandra

ADD files/sshd_config /var/lib/cassandra/custom_ssh/
ADD files/run.sh /

EXPOSE 7000 7001 7199 9042 9160 2222 8778

RUN chmod +x /run.sh
