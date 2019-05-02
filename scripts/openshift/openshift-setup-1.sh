#!/usr/bin/env bash

set -Eeuxo pipefail

# This script is part one of setting up openshift test environment. At the end
# of this script, a shell re-login is needed to be able to use docker as a
# non-root user. Due to this, there's a second part of this script which
# does the openshift setup.

# Update and install git. Git it required to get the operator source code.
sudo apt -y update
sudo apt -y install git

# Install docker
sudo wget -qO- https://get.docker.com/ | sh
sudo usermod -aG docker $USER

# Downgrade to docker 18.06.1 on trusty because the latest version fails to
# start container due to a security fix which requires certain kernel featues
# that are not available in trusty's default kernel.
# https://github.com/docker/for-linux/issues/591
sudo apt install -y docker-ce=18.06.1~ce~3-0~ubuntu --force-yes

# Set docker insecure registry to install openshift.
sudo service docker stop
sudo tee /etc/docker/daemon.json > /dev/null <<EOF
{
    "insecure-registries" : ["172.30.0.0/16"]
}
EOF
sudo service docker start
