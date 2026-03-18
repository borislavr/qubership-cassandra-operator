#!/bin/bash
set -x
set -e
CASSANDRA_CONFIG=$CASSANDRA_CONFIG_DIR/cassandra.yaml

cp --remove-destination ${CASSANDRA_INIT_CONFIG_DIR}/* ${CASSANDRA_CONFIG_DIR}/

CASSANDRA_DATA=/var/lib/cassandra/data

if [[ -z "$HOST_NAME" ]]; then
  HOST_NAME=$(ip addr | grep 'BROADCAST' -A2 | tail -n1 | awk '{print $2}' | cut -f1  -d'/')
fi
CASSANDRA_ENDPOINT_SNITCH="${CASSANDRA_ENDPOINT_SNITCH:-SimpleSnitch}"
CASSANDRA_DC="${CASSANDRA_DC}"
CASSANDRA_RACK="${CASSANDRA_RACK}"
CASSANDRA_LISTEN_ADDRESS="${CASSANDRA_LISTEN_ADDRESS:-${POD_IP:-$HOST_NAME}}"
CASSANDRA_BROADCAST_ADDRESS="${CASSANDRA_BROADCAST_ADDRESS:-${POD_IP:-$HOST_NAME}}"
CASSANDRA_RPC_ADDRESS="${CASSANDRA_RPC_ADDRESS:-0.0.0.0}"
CASSANDRA_BROADCAST_RPC_ADDRESS="${CASSANDRA_BROADCAST_RPC_ADDRESS:-${POD_IP:-$HOST_NAME}}"
PRERESOLVE_SEEDS="${PRERESOLVE_SEEDS:-false}"


if [[ $PRERESOLVE_SEEDS == "true" ]]; then
  IFS=', ' read -r -a array <<< "$CASSANDRA_SEEDS"

  CASSANDRA_SEEDS_IP=""
  for element in "${array[@]}"
  do
    ip=""
    ip=$(nslookup "$element" | grep -e "Address:.*10\." | awk '{print $2}')
    if [[ ! $ip ]]; then
      ip="$element"
    fi
    if [[ $CASSANDRA_SEEDS_IP ]]; then
      CASSANDRA_SEEDS_IP="$CASSANDRA_SEEDS_IP, $ip"
    else
      CASSANDRA_SEEDS_IP="$ip"
    fi
  done

  CASSANDRA_SEEDS="$CASSANDRA_SEEDS_IP"
fi

echo $CASSANDRA_SEEDS
echo $CASSANDRA_CONFIG


sed -ri 's/POD_IP/'$POD_IP'/' $CASSANDRA_CONFIG_DIR/cassandra-env.sh

sed -ri 's/- seeds:.*/- seeds: "'"$CASSANDRA_SEEDS"'"/' $CASSANDRA_CONFIG

cat $CASSANDRA_CONFIG_DIR/cassandra.yaml | grep "seeds:"

# if DC and RACK are set, use GossipingPropertyFileSnitch
if [[ $CASSANDRA_DC && $CASSANDRA_RACK ]]; then
  echo "dc=$CASSANDRA_DC" > $CASSANDRA_CONFIG_DIR/cassandra-rackdc.properties
  echo "rack=$CASSANDRA_RACK" >> $CASSANDRA_CONFIG_DIR/cassandra-rackdc.properties
  # it is necessary to broadcast_address will reroute to listen_address in local network
  echo "prefer_local=true" >> $CASSANDRA_CONFIG_DIR/cassandra-rackdc.properties
  CASSANDRA_ENDPOINT_SNITCH="GossipingPropertyFileSnitch"
fi
for yaml in \
  broadcast_address \
  broadcast_rpc_address \
  rpc_address \
  endpoint_snitch \
  listen_address \
  ; do
  var="CASSANDRA_${yaml^^}"
  val1="${!var}"
  val=$(echo $val1 | sed 's_/_\\/_g')
  if [ "$val" ]; then
    sed -ri 's/^(# )?('"$yaml"':).*/\2 '"$val"'/' "$CASSANDRA_CONFIG"
  fi
done


rm -f $CASSANDRA_CONFIG_DIR/cassandra-topology.properties


if [[ $TLS == "true" ]]; then
  rm -f /var/lib/cassandra/data/keystore.p12 && openssl pkcs12 -export -in $TLS_SIGNED -inkey $TLS_KEY -CAfile $TLS_CA -name "$HOST_NAME" -out /var/lib/cassandra/data/keystore.p12 -password pass:$TLS_PASS
fi


if ! whoami &>/dev/null; then
    echo "cassandra:x:$(id -u):$(id -g):cassandra user:${CASSANDRA_HOME}:/bin/bash" >> /etc/passwd
fi


yes y | ssh-keygen -f /var/lib/cassandra/custom_ssh/ssh_host_rsa_key -N '' -t rsa
/usr/sbin/sshd -f /var/lib/cassandra/custom_ssh/sshd_config &
cassandra -f "$COMMAND_PARAMS"
