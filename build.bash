#!/bin/bash
set -eo pipefail
if [ ! -z "${BASH_VERBOSE}" ]; then
    set -x
fi

export GOPATH=${TEAMCITY_CHECKOUT_DIR}
pushd ${TEAMCITY_CHECKOUT_DIR}/src/github.com/platform9/pf9ctl
   make container-build
popd

mkdir -p ${TEAMCITY_CHECKOUT_DIR}/build/publish-to-artf
cat > ${TEAMCITY_CHECKOUT_DIR}/build/upload_spec.json << EOF
{
  "files": [
    {
      "pattern": "build/publish-to-artf/pf9ctl",
      "target": "pf9-bins/pf9-ctl/",
      "flat": true
    }
  ]
}
EOF

echo "##teamcity[setParameter name='env.UPLOAD_SPEC_PATH' value='${TEAMCITY_CHECKOUT_DIR}/build/upload_spec.json']"

if [ ${TEAMCITY_BUILD_BRANCH} = master ]; then
    ln ${TEAMCITY_CHECKOUT_DIR}/src/github.com/platform9/pf9ctl/bin/pf9ctl ${TEAMCITY_CHECKOUT_DIR}/build/publish-to-artf/pf9ctl
else
    echo "Not publishing pf9ctl binary to artifactory since this is not built off master branch."
fi
