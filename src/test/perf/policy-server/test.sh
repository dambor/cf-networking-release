# !/bin/bash

set -e -u
set -o pipefail

export API="https://api.bosh-lite.com"
export CF_USER=admin
export CF_PASSWORD=admin

cd $GOPATH

go run src/test/perf/policy-server/main.go \
	-apps 10 \
	-numCells 1 \
	-policiesPerApp 3 \
	-pollInterval 5s \
	-cfUser "$CF_USER" \
	-cfPassword "$CF_PASSWORD" \
	-api "$API" \
	-setup=false
	
