#!/bin/bash

# This script created a Prometheus binary to use in benchmarks and place it in $VOLUME/prometheus.
# It uses $REFERENCE variable to build or fetch the binary.
# If $USE_REGISTRY is set to "true" it uses the binary from the quay.io/prometheus/prometheus:${REFERENCE},
# otherwise it builds it from source, from the given reference (PR number, branch or git SHA).

# Default values
DIR="/go/src/github.com/prometheus/prometheus"

REFERENCE=${REFERENCE}

# We want builder to work with the old scenarios, so support PR_NUMBER var for compatibility.
if [[ -z ${REFERENCE} ]]; then
  REFERENCE=${PR_NUMBER}
fi

if [[ -z ${REFERENCE} || -z ${VOLUME_DIR} || -z ${GITHUB_ORG} || -z ${GITHUB_REPO} ]]; then
    echo "ERROR:: environment variables not set correctly, requires REFERENCE (or PR_NUMBER), VOLUME_DIR, GITHUB_ORG, GITHUB_REPO"
    exit 1;
fi

# Fetch from quay if requested.
if [[ "${USE_REGISTRY}" == "true" ]]; then
    echo ">> USE_PRE_BUILD is enabled."
    echo ">> Attempting to download binary from quay.io/prometheus/prometheus:${REFERENCE}"

    IMAGE="quay.io/prometheus/prometheus:${REFERENCE}"

    if ! CONTAINER_ID=$(docker create "${IMAGE}"); then
        echo "ERROR:: Could not pull or create container from ${IMAGE}"
        exit 1
    fi

    echo ">> Extracting prometheus binary from ${CONTAINER_ID} container..."
    if ! docker cp "${CONTAINER_ID}:/bin/prometheus" "${VOLUME_DIR}/prometheus"; then
        echo "ERROR:: Failed to copy binary from container"
        docker rm -v "${CONTAINER_ID}" >/dev/null
        exit 1
    fi
    docker rm -v "${CONTAINER_ID}" >/dev/null
    echo ">> Binary successfully downloaded and copied."
    exit 0
fi

# Clone the repository with a shallow clone
echo ">> Cloning repository $GITHUB_ORG/$GITHUB_REPO (shallow clone)"
if ! git clone --depth 1 "https://github.com/$GITHUB_ORG/$GITHUB_REPO.git" "$DIR"; then
    echo "ERROR:: Cloning of repo $GITHUB_ORG/$GITHUB_REPO failed"
    exit 1;
fi

cd "$DIR" || exit 1

echo ">> Resolving git state for building from source from ${REFERENCE}..."

# Attempt 1: Try pulling PR first which will only work if REFERENCE is a PR number.
if git fetch origin "pull/${REFERENCE}/head:pr-branch" 2>/dev/null; then
    echo ">> Successfully fetched PR reference: pull/${REFERENCE}/head"
    git checkout pr-branch
else
    # Attempt 2: If PR fetch fails, try assuming it's a branch or Git SHA
    echo ">> Reference 'pull/${REFERENCE}/head' not found; assuming reference is not a PR number. Trying to fetch '${REFERENCE}' as a remote branch or SHA..."

    # We fetch specifically the ref to FETCH_HEAD to avoid naming conflicts
    if git fetch origin "${REFERENCE}"; then
        echo ">> Successfully fetched reference: ${REFERENCE}"
        git checkout FETCH_HEAD
    else
        echo "ERROR:: Could not resolve '${REFERENCE}' as a Pull Request, remote branch, or Git SHA."
        exit 1
    fi
fi

echo ">> Creating prometheus binary using promu"
if ! make build PROMU_BINARIES="prometheus"; then
    echo "ERROR:: Building of binaries failed"
    exit 1;
fi

echo ">> Copy files to volume"
cp prometheus "$VOLUME_DIR/prometheus"