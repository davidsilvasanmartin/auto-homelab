#!/bin/sh

# See https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#password-reset
HOMELAB_ADGUARD_PASSWORD_HASHED=$(htpasswd -B -C 10 -n -b admin ${HOMELAB_ADGUARD_PASSWORD} | cut -d: -f2-)

# Replace environment variables in the template file.
sed -e "s|\${HOMELAB_GENERAL_DOMAIN}|${HOMELAB_GENERAL_DOMAIN}|g" \
    -e "s|\${HOMELAB_GENERAL_SERVER_IP}|${HOMELAB_GENERAL_SERVER_IP}|g" \
    -e "s|\${HOMELAB_ADGUARD_PASSWORD_HASHED}|${HOMELAB_ADGUARD_PASSWORD_HASHED}|g" \
    < /auto-homelab/scripts/AdGuardHome.yaml.template > /opt/adguardhome/conf/AdGuardHome.yaml

# Check if the config file was created successfully
if [ ! -s /opt/adguardhome/conf/AdGuardHome.yaml ]; then
    echo "Error: AdGuardHome.yaml is empty after variable substitution. Exiting."
    exit 1
fi

# Execute the original adguard command. I found this command by examining the original Dockerfile.
# I may have to review and change this command in the future.
exec /opt/adguardhome/AdGuardHome --no-check-update -c /opt/adguardhome/conf/AdGuardHome.yaml -w /opt/adguardhome/work
