#!/bin/bash
set -e -u

cd "$(dirname "$0")"
#(cd ../uplink-webapp && make node_modules && npm run build)
make build
make package
echo "now scp dist/mattermost-enterprise-linux-amd64.tar.gz over to uplink.mit.edu"
scp dist/mattermost-enterprise-linux-amd64.tar.gz uplink-common@uplink.mit.edu:~
ssh -t uplink-common@uplink.mit.edu ./deploy.sh
