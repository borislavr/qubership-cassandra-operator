#!/bin/bash
set -x
set -e


if [[ $REAPER_CASS_NATIVE_PROTOCOL_SSL_ENCRYPTION_ENABLED == "true" ]]; then
  keytool -import -trustcacerts -noprompt -alias ca_alias -file /usr/ssl/ca.crt -keystore /tmp/truststore.jks -storepass reaper_password
  openssl pkcs12 -export -in /usr/ssl/tls.crt -inkey /usr/ssl/tls.key -out /tmp/client.p12 -CAfile /usr/ssl/ca.crt -name client-cert -passout pass:reaper_password
fi

cp --remove-destination /etc/cassandra-reaper-temp/cassandra-reaper.yml /etc/cassandra-reaper/config/
#call original entrypoint
exec /usr/local/bin/entrypoint.sh cassandra-reaper