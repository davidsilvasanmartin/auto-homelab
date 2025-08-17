#!/bin/sh

# TODO !!!

# We use `sed` to replace environment variables in the template file.
sed -e "s|\${HOMELAB_GENERAL_DOMAIN}|${HOMELAB_GENERAL_DOMAIN}|g" \
    -e "s|\${HOMELAB_GENERAL_SERVER_IP}|${HOMELAB_GENERAL_SERVER_IP}|g" \
    < /opt/unbound/etc/unbound/unbound.conf.template > /opt/unbound/etc/unbound/unbound.conf

# Check if the config file was created successfully
if [ ! -s /opt/unbound/etc/unbound/unbound.conf ]; then
    echo "Error: unbound.conf is empty after variable substitution. Exiting."
    exit 1
fi

# Execute the original unbound command
exec /opt/unbound/sbin/unbound -d -c /opt/unbound/etc/unbound/unbound.conf
