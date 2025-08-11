#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2024 Red Hat, Inc.
#

set -e

# Build kube-api-linter plugin if not exists or if source is newer
PLUGIN_DIR="_out/linter"
PLUGIN_PATH="${PLUGIN_DIR}/kube-api-linter.so"

mkdir -p ${PLUGIN_DIR}

# Check if we need to build/rebuild the plugin
NEED_BUILD=false

if [ ! -f "${PLUGIN_PATH}" ]; then
  echo "Building kube-api-linter plugin..."
  NEED_BUILD=true
elif command -v go >/dev/null 2>&1; then
  # Check if go.mod is newer than the plugin
  if [ go.mod -nt "${PLUGIN_PATH}" ] || [ go.sum -nt "${PLUGIN_PATH}" ]; then
    echo "Dependencies changed, rebuilding kube-api-linter plugin..."
    NEED_BUILD=true
  fi
fi

if [ "$NEED_BUILD" = true ]; then
  # Temporarily disable workspace mode for plugin building
  export GOWORK=off

  # Check if kube-api-linter dependency is available
  if ! go list -m sigs.k8s.io/kube-api-linter >/dev/null 2>&1; then
    echo "Adding kube-api-linter dependency..."
    go get sigs.k8s.io/kube-api-linter@latest
  fi

  # Ensure CGO is enabled for plugin building
  export CGO_ENABLED=1
  go build -mod=mod -buildmode=plugin -o "${PLUGIN_PATH}" sigs.k8s.io/kube-api-linter/pkg/plugin
  echo "kube-api-linter plugin built successfully"

  # Re-enable workspace mode
  unset GOWORK
fi

paths=""
while IFS= read -r line; do
  # read directory from the file and append a wildcard
  paths+="${line}/... "
done <hack/linter/lint-paths.txt

golangci-lint run --timeout 20m --verbose ${paths}
golangci-lint run --default=none --enable=ginkgolinter --timeout 10m --verbose --no-config \
  ./pkg/... \
  ./tests/...
