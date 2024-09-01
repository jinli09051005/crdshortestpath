
#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail
GOPATH=$(go env GOPATH)
SCRIPT=$(dirname "${BASH_SOURCE[0]}")
SCRIPT_ROOT="${SCRIPT}"/..
CODEGEN_PKG=${CODEGEN_PKG:-$(echo ../code-generator)}

if [ "$1" == "external" ]; then
bash "${CODEGEN_PKG}"/generate-groups.sh " \
deepcopy,client,lister,informer" \
 $2/generated/external \
 $2/$3 \
 $4 \
 --output-base "${GOPATH}/src" \
 --go-header-file "${SCRIPT}"/boilerplate.go.txt
fi
if [ "$1" == "internal" ]; then
GOPATH="${GOPATH}" bash "${CODEGEN_PKG}"/generate-internal-groups.sh \
"deepcopy,client,lister,informer,conversion,defaulter" \
 $2/generated/client \
 $2/$3 \
 $2/$3 \
 $4 \
 --output-base "${GOPATH}/src" \
 --go-header-file "${SCRIPT}"/boilerplate.go.txt
GOPATH="${GOPATH}" bash "${CODEGEN_PKG}"/generate-internal-groups.sh \
"openapi" \
 $2/generated \
 $2/$3 \
 $2/$3 \
 $4 \
 --output-base "${GOPATH}/src" \
 --go-header-file "${SCRIPT}"/boilerplate.go.txt
fi
