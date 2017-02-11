# !/bin/bash

set -e -u
set -o pipefail

export API="https://api.bosh-lite.com"
cf api "$API" --skip-ssl-validation
cf auth admin admin
export TOKEN=`cf oauth-token`

cd $GOPATH

go run src/test/perf/policy-server/main.go \
	-apps 10000 \
	-numCells 100 \
	-policiesPerApp 3 \
	-pollInterval 5s \
	-token "$TOKEN" \
	-api "$API" \
	-setup=false
	
