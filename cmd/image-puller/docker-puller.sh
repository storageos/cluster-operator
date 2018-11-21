#!/bin/ash

set -euo pipefail

# Gracefully handle the TERM signal sent when deleting the daemonset
trap 'exit' TERM

# This is the main command that's run by this script on
# all the nodes.
docker pull $1

# Let the monitoring script know we're done.
echo "done"

# this is a workaround to prevent the container from exiting
# and k8s restarting the daemonset pod
while true; do sleep 1; done
