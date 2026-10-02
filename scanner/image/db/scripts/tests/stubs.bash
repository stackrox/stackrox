#!/usr/bin/env bash

# Stubs for the PostgreSQL binaries that init-entrypoint.sh drives, so the
# upgrade logic can be exercised without a real cluster.
#
# Every stub appends its name and arguments to $STUB_LOG, so a test can assert
# which branch was taken by what was (not) invoked. Behaviour is steered by
# STUB_* variables, which the stubs read at call time from $STUB_ENV rather
# than inheriting, so a test can change them after setup().
#
# Defaults describe the happy path: binaries are PostgreSQL 16, the cluster is
# cleanly shut down, every pg_upgrade invocation succeeds, and there is plenty
# of free space.

# stub_setup creates the stub binaries and points init-entrypoint.sh at them.
# It must be called before sourcing init-entrypoint.sh.
stub_setup() {
    STUB_ROOT="${BATS_TEST_TMPDIR}/stub"
    STUB_BIN="${STUB_ROOT}/usr/bin"
    STUB_OLD_ROOT="${STUB_ROOT}/usr/lib64/pgsql"
    STUB_LOG="${BATS_TEST_TMPDIR}/stub.log"
    STUB_ENV="${BATS_TEST_TMPDIR}/stub.env"

    export STUB_LOG STUB_ENV

    mkdir -p "$STUB_BIN" "${STUB_OLD_ROOT}/postgresql-15/bin"
    : >"$STUB_LOG"
    : >"$STUB_ENV"

    stub_set STUB_BINARY_VERSION 16
    stub_set STUB_CLUSTER_STATE "shut down"
    stub_set STUB_INITDB_RC 0
    stub_set STUB_PG_CTL_RC 0
    stub_set STUB_PG_ISREADY_RC 0
    stub_set STUB_CLONE_CHECK_RC 0
    stub_set STUB_COPY_CHECK_RC 0
    stub_set STUB_UPGRADE_RC 0
    stub_set STUB_VACUUMDB_RC 0
    stub_set STUB_DU_KB 1000
    stub_set STUB_DF_AVAIL_KB 1000000

    _stub_write postgres <<'EOF'
if [ "${1:-}" = "-V" ]; then
    echo "postgres (PostgreSQL) ${STUB_BINARY_VERSION}.2"
fi
EOF

    _stub_write initdb <<'EOF'
datadir=""
while [ "$#" -gt 0 ]; do
    case "$1" in
        --pgdata) datadir="$2"; shift 2 ;;
        --pgdata=*) datadir="${1#--pgdata=}"; shift ;;
        *) shift ;;
    esac
done
[ -n "$datadir" ] || { echo "initdb: no --pgdata" >&2; exit 1; }
if [ "$STUB_INITDB_RC" != 0 ]; then
    # The real initdb leaves a partial directory behind when it runs out of
    # room part way through.
    mkdir -p "$datadir"
    echo "initdb: could not write to $datadir: No space left on device" >&2
    exit "$STUB_INITDB_RC"
fi
mkdir -p "$datadir"
echo "$STUB_BINARY_VERSION" >"$datadir/PG_VERSION"
echo "initdb" >"$datadir/postgresql.conf"
EOF

    # pg_ctl and pg_isready are invoked both through PATH (new binaries, by
    # analyze-cluster) and by absolute path out of the old bindir (by
    # clean-shutdown), so the same stub is installed in both places.
    _stub_write pg_ctl <<'EOF'
exit "$STUB_PG_CTL_RC"
EOF

    _stub_write pg_isready <<'EOF'
exit "$STUB_PG_ISREADY_RC"
EOF

    _stub_write pg_controldata <<'EOF'
echo "pg_control version number:            1300"
echo "Database cluster state:               ${STUB_CLUSTER_STATE}"
EOF

    # Mirrors the real thing closely enough for the tests that matter: --check
    # never touches either directory, and a real run materializes the upgraded
    # cluster in --new-datadir. Neither ever modifies --old-datadir, which is
    # the property the whole no-abort design rests on.
    _stub_write pg_upgrade <<'EOF'
old_datadir="" new_datadir="" check=false mode=""
for arg in "$@"; do
    case "$arg" in
        --old-datadir=*) old_datadir="${arg#--old-datadir=}" ;;
        --new-datadir=*) new_datadir="${arg#--new-datadir=}" ;;
        --check) check=true ;;
        --clone|--copy|--link) mode="$arg" ;;
    esac
done

if [ "$check" = true ]; then
    case "$mode" in
        --clone) exit "$STUB_CLONE_CHECK_RC" ;;
        --copy)  exit "$STUB_COPY_CHECK_RC" ;;
        *)       exit 0 ;;
    esac
fi

if [ "$STUB_UPGRADE_RC" != 0 ]; then
    # The real pg_upgrade leaves its server log behind on failure.
    mkdir -p "$new_datadir/pg_upgrade_output.d/stub"
    echo "stub pg_upgrade failure" >"$new_datadir/pg_upgrade_output.d/stub/pg_upgrade_server.log"
    exit "$STUB_UPGRADE_RC"
fi

# The real pg_upgrade transfers user data into the new cluster but leaves
# its initdb-generated configuration alone, and writes no sentinel -- which
# is exactly why init-entrypoint.sh has to touch .init-ready itself.
cp -R "$old_datadir/base" "$new_datadir/"
echo "$STUB_BINARY_VERSION" >"$new_datadir/PG_VERSION"
echo "Upgrade Complete"
EOF

    _stub_write vacuumdb <<'EOF'
exit "$STUB_VACUUMDB_RC"
EOF

    _stub_write du <<'EOF'
echo "${STUB_DU_KB}	${*: -1}"
EOF

    _stub_write df <<'EOF'
echo "Filesystem 1024-blocks Used Available Capacity Mounted on"
echo "/dev/stub 2000000 1 ${STUB_DF_AVAIL_KB} 1% /"
EOF

    cp "$STUB_BIN/pg_ctl" "$STUB_BIN/pg_isready" "$STUB_BIN/pg_controldata" \
        "${STUB_OLD_ROOT}/postgresql-15/bin/"

    PATH="${STUB_BIN}:${PATH}"
    export PATH
}

# stub_set records a STUB_* value. Stubs read these at call time, so this
# works both before and after the script under test is sourced.
stub_set() {
    local name=${1:?missing first required argument: name}
    local value=${2?missing second required argument: value}

    # Last assignment wins when the stub sources the file.
    printf '%s=%q\n' "$name" "$value" >>"$STUB_ENV"
}

# stub_log echoes everything the stubs recorded.
stub_log() {
    cat "$STUB_LOG"
}

# assert_stub_called fails unless some stub invocation matched the given
# extended regular expression.
assert_stub_called() {
    local pattern=${1:?missing first required argument: pattern}

    if ! grep -Eq -- "$pattern" "$STUB_LOG"; then
        echo "expected a stub invocation matching: $pattern"
        echo "--- stub invocations ---"
        cat "$STUB_LOG"
        return 1
    fi
}

# refute_stub_called fails if any stub invocation matched the given extended
# regular expression.
refute_stub_called() {
    local pattern=${1:?missing first required argument: pattern}

    if grep -Eq -- "$pattern" "$STUB_LOG"; then
        echo "expected no stub invocation matching: $pattern"
        echo "--- stub invocations ---"
        cat "$STUB_LOG"
        return 1
    fi
}

_stub_write() {
    local name=${1:?missing first required argument: name}

    {
        echo '#!/usr/bin/env bash'
        echo 'set -eo pipefail'
        # shellcheck disable=SC2016  # expanded by the stub, not here.
        echo 'source "$STUB_ENV"'
        echo "echo \"$name \$*\" >>\"\$STUB_LOG\""
        cat
    } >"$STUB_BIN/$name"
    chmod +x "$STUB_BIN/$name"
}
