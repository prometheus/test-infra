#!/usr/bin/env bash
#
# Generate --help output for all commands and embed them into the respective docs.
set -e
set -u

EMBEDMD_BIN=${EMBEDMD_BIN:-embedmd}
SED_BIN=${SED_BIN:-sed}

README_FILES="./tools/*/README.md ./infra/README.md"

primary_tools=("infra")
helper_tools=("amGithubNotifier" "commentMonitor")

function fetch_embedmd {
  pushd ..
  go install github.com/campoy/embedmd/v2@latest
  popd
}

function docs {
# If check arg was passed, instead of the docs generation verifies if docs coincide with the codebase.
if [[ "${CHECK}" == "check" ]]; then
    set +e
    DIFF=$(${EMBEDMD_BIN} -d ${README_FILES})
    RESULT=$?
    if [[ "$RESULT" != "0" ]]; then
        cat << EOF
Docs have discrepancies, do 'make build docs' and commit changes:

${DIFF}
EOF
        exit 2
fi
else
    ${EMBEDMD_BIN} -w ${README_FILES}
fi
}

if ! [[ "$0" =~ "scripts/genflagdocs.sh" ]]; then
	echo "must be run from repository root"
	exit 255
fi

CHECK=${1:-}

# Auto update flags and remove white noise.
for x in "${primary_tools[@]}"; do
    "./${x}/${x}" --help &> "./${x}/${x}-flags.txt"
    ${SED_BIN} -i -e 's/[ \t]*$//' "./${x}/${x}-flags.txt"
done

for x in "${helper_tools[@]}"; do
    "./tools/${x}/${x}" --help &> "./tools/${x}/${x}-flags.txt"
    ${SED_BIN} -i -e 's/[ \t]*$//' "./tools/${x}/${x}-flags.txt"
done

fetch_embedmd
docs
