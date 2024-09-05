#!/usr/bin/env bash
# This file is from https://github.com/docker-library/postgres/tree/master/14/bullseye
set -Eeo pipefail
# TODO swap to -Eeuo pipefail above (after handling all potentially-unset variables)

shutdown() {
  pg_ctl -D /var/lib/postgresql/data/pgdata stop -m fast
}

trap shutdown SIGINT SIGTERM

# usage: file_env VAR [DEFAULT]
#    ie: file_env 'XYZ_DB_PASSWORD' 'example'
# (will allow for "$XYZ_DB_PASSWORD_FILE" to fill in the value of
#  "$XYZ_DB_PASSWORD" from a file, especially for Docker's secrets feature)
file_env() {
	local var="$1"
	local fileVar="${var}_FILE"
	local def="${2:-}"
	if [ "${!var:-}" ] && [ "${!fileVar:-}" ]; then
		echo >&2 "error: both $var and $fileVar are set (but are exclusive)"
		exit 1
	fi
	local val="$def"
	if [ "${!var:-}" ]; then
		val="${!var}"
	elif [ "${!fileVar:-}" ]; then
		val="$(< "${!fileVar}")"
	fi
	export "$var"="$val"
	unset "$fileVar"
}

# check to see if this file is being run or sourced from another script
_is_sourced() {
	# https://unix.stackexchange.com/a/215279
	[ "${#FUNCNAME[@]}" -ge 2 ] \
		&& [ "${FUNCNAME[0]}" = '_is_sourced' ] \
		&& [ "${FUNCNAME[1]}" = 'source' ]
}

# used to create initial postgres directories and if run as root, ensure ownership to the "postgres" user
docker_create_db_directories() {
	local user; user="$(id -u)"

	mkdir -p "$PGDATA"
	# ignore failure since there are cases where we can't chmod (and PostgreSQL might fail later anyhow - it's picky about permissions of this directory)
	chmod 700 "$PGDATA" || :

	# ignore failure since it will be fine when using the image provided directory; see also https://github.com/docker-library/postgres/pull/289
	mkdir -p /var/run/postgresql || :
	chmod 775 /var/run/postgresql || :

	# Create the transaction log directory before initdb is run so the directory is owned by the correct user
	if [ -n "$POSTGRES_INITDB_WALDIR" ]; then
		mkdir -p "$POSTGRES_INITDB_WALDIR"
		if [ "$user" = '0' ]; then
			find "$POSTGRES_INITDB_WALDIR" \! -user postgres -exec chown postgres '{}' +
		fi
		chmod 700 "$POSTGRES_INITDB_WALDIR"
	fi

	# allow the container to be started with `--user`
	if [ "$user" = '0' ]; then
		find "$PGDATA" \! -user postgres -exec chown postgres '{}' +
		find /var/run/postgresql \! -user postgres -exec chown postgres '{}' +
	fi
}

# initialize empty PGDATA directory with new database via 'initdb'
# arguments to `initdb` can be passed via POSTGRES_INITDB_ARGS or as arguments to this function
# `initdb` automatically creates the "postgres", "template0", and "template1" dbnames
# this is also where the database user is created, specified by `POSTGRES_USER` env
docker_init_database_dir() {
	# "initdb" is particular about the current user existing in "/etc/passwd", so we use "nss_wrapper" to fake that if necessary
	# see https://github.com/docker-library/postgres/pull/253, https://github.com/docker-library/postgres/issues/359, https://cwrap.org/nss_wrapper.html
	local uid; uid="$(id -u)"
	if ! getent passwd "$uid" &> /dev/null; then
		# see if we can find a suitable "libnss_wrapper.so" (https://salsa.debian.org/sssd-team/nss-wrapper/-/commit/b9925a653a54e24d09d9b498a2d913729f7abb15)
		local wrapper
		for wrapper in {/usr,}/lib{/*,}/libnss_wrapper.so; do
			if [ -s "$wrapper" ]; then
				NSS_WRAPPER_PASSWD="$(mktemp)"
				NSS_WRAPPER_GROUP="$(mktemp)"
				export LD_PRELOAD="$wrapper" NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP
				local gid; gid="$(id -g)"
				echo "postgres:x:$uid:$gid:PostgreSQL:$PGDATA:/bin/false" > "$NSS_WRAPPER_PASSWD"
				echo "postgres:x:$gid:" > "$NSS_WRAPPER_GROUP"
				break
			fi
		done
	fi

	if [ -n "$POSTGRES_INITDB_WALDIR" ]; then
		set -- --waldir "$POSTGRES_INITDB_WALDIR" "$@"
	fi

	eval 'initdb --username="$POSTGRES_USER" --pwfile=<(echo "$POSTGRES_PASSWORD") --encoding=UTF8 '"$POSTGRES_INITDB_ARGS"' "$@"'

	# unset/cleanup "nss_wrapper" bits
	if [[ "${LD_PRELOAD:-}" == */libnss_wrapper.so ]]; then
		rm -f "$NSS_WRAPPER_PASSWD" "$NSS_WRAPPER_GROUP"
		unset LD_PRELOAD NSS_WRAPPER_PASSWD NSS_WRAPPER_GROUP
	fi
}

# print large warning if POSTGRES_PASSWORD is long
# error if both POSTGRES_PASSWORD is empty and POSTGRES_HOST_AUTH_METHOD is not 'trust'
# print large warning if POSTGRES_HOST_AUTH_METHOD is set to 'trust'
# assumes database is not set up, ie: [ -z "$DATABASE_ALREADY_EXISTS" ]
docker_verify_minimum_env() {
	# check password first so we can output the warning before postgres
	# messes it up
	if [ "${#POSTGRES_PASSWORD}" -ge 100 ]; then
		cat >&2 <<-'EOWARN'

			WARNING: The supplied POSTGRES_PASSWORD is 100+ characters.

			  This will not work if used via PGPASSWORD with "psql".

			  https://www.postgresql.org/message-id/flat/E1Rqxp2-0004Qt-PL%40wrigleys.postgresql.org (BUG #6412)
			  https://github.com/docker-library/postgres/issues/507

		EOWARN
	fi
	if [ -z "$POSTGRES_PASSWORD" ] && [ 'trust' != "$POSTGRES_HOST_AUTH_METHOD" ]; then
		# The - option suppresses leading tabs but *not* spaces. :)
		cat >&2 <<-'EOE'
			Error: Database is uninitialized and superuser password is not specified.
			       You must specify POSTGRES_PASSWORD to a non-empty value for the
			       superuser. For example, "-e POSTGRES_PASSWORD=password" on "docker run".

			       You may also use "POSTGRES_HOST_AUTH_METHOD=trust" to allow all
			       connections without a password. This is *not* recommended.

			       See PostgreSQL documentation about "trust":
			       https://www.postgresql.org/docs/current/auth-trust.html
		EOE
		exit 1
	fi
	if [ 'trust' = "$POSTGRES_HOST_AUTH_METHOD" ]; then
		cat >&2 <<-'EOWARN'
			********************************************************************************
			WARNING: POSTGRES_HOST_AUTH_METHOD has been set to "trust". This will allow
			         anyone with access to the Postgres port to access your database without
			         a password, even if POSTGRES_PASSWORD is set. See PostgreSQL
			         documentation about "trust":
			         https://www.postgresql.org/docs/current/auth-trust.html
			         In Docker's default configuration, this is effectively any other
			         container on the same system.

			         It is not recommended to use POSTGRES_HOST_AUTH_METHOD=trust. Replace
			         it with "-e POSTGRES_PASSWORD=password" instead to set a password in
			         "docker run".
			********************************************************************************
		EOWARN
	fi
}

# usage: docker_process_init_files [file [file [...]]]
#    ie: docker_process_init_files /always-initdb.d/*
# process initializer files, based on file extensions and permissions
docker_process_init_files() {
	# psql here for backwards compatibility "${psql[@]}"
	psql=( docker_process_sql )

	echo
	local f
	for f; do
		case "$f" in
			*.sh)
				# https://github.com/docker-library/postgres/issues/450#issuecomment-393167936
				# https://github.com/docker-library/postgres/pull/452
				if [ -x "$f" ]; then
					echo "$0: running $f"
					"$f"
				else
					echo "$0: sourcing $f"
					. "$f"
				fi
				;;
			*.sql)    echo "$0: running $f"; docker_process_sql -f "$f"; echo ;;
			*.sql.gz) echo "$0: running $f"; gunzip -c "$f" | docker_process_sql; echo ;;
			*.sql.xz) echo "$0: running $f"; xzcat "$f" | docker_process_sql; echo ;;
            *scl_enable)
                if [ -x "$f" ]; then
                    echo "$0: running $f"
                    "$f"
                else
                    echo "$0: sourcing $f"
                    . "$f"
                fi
                ;;
			*)        echo "$0: ignoring $f" ;;
		esac
		echo
	done
}

# Execute sql script, passed via stdin (or -f flag of pqsl)
# usage: docker_process_sql [psql-cli-args]
#    ie: docker_process_sql --dbname=mydb <<<'INSERT ...'
#    ie: docker_process_sql -f my-file.sql
#    ie: docker_process_sql <my-file.sql
docker_process_sql() {
	local query_runner=( psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --no-password --no-psqlrc )
	if [ -n "$POSTGRES_DB" ]; then
		query_runner+=( --dbname "$POSTGRES_DB" )
	fi

	PGHOST= PGHOSTADDR= "${query_runner[@]}" "$@"
}

# create initial database
# uses environment variables for input: POSTGRES_DB
docker_setup_db() {
	local dbAlreadyExists
	dbAlreadyExists="$(
		POSTGRES_DB= docker_process_sql --dbname postgres --set db="$POSTGRES_DB" --tuples-only <<-'EOSQL'
			SELECT 1 FROM pg_database WHERE datname = :'db' ;
		EOSQL
	)"
	if [ -z "$dbAlreadyExists" ]; then
		POSTGRES_DB= docker_process_sql --dbname postgres --set db="$POSTGRES_DB" <<-'EOSQL'
			CREATE DATABASE :"db" ;
		EOSQL
		echo
	fi
}

# Loads various settings that are used elsewhere in the script
# This should be called before any other functions
docker_setup_env() {
	file_env 'POSTGRES_PASSWORD'

	file_env 'POSTGRES_USER' 'postgres'
	file_env 'POSTGRES_DB' "$POSTGRES_USER"
	file_env 'POSTGRES_INITDB_ARGS'
	: "${POSTGRES_HOST_AUTH_METHOD:=}"

	declare -g DATABASE_ALREADY_EXISTS
	# look specifically for PG_VERSION, as it is expected in the DB dir
	if [ -s "$PGDATA/PG_VERSION" ]; then
		DATABASE_ALREADY_EXISTS='true'
	fi
}

# append POSTGRES_HOST_AUTH_METHOD to pg_hba.conf for "host" connections
# all arguments will be passed along as arguments to `postgres` for getting the value of 'password_encryption'
pg_setup_hba_conf() {
	# default authentication method is md5 on versions before 14
	# https://www.postgresql.org/about/news/postgresql-14-released-2318/
	if [ "$1" = 'postgres' ]; then
		shift
	fi
	local auth
	# check the default/configured encryption and use that as the auth method
	auth="$(postgres -C password_encryption "$@")"
	: "${POSTGRES_HOST_AUTH_METHOD:=$auth}"
	{
		echo
		if [ 'trust' = "$POSTGRES_HOST_AUTH_METHOD" ]; then
			echo '# warning trust is enabled for all connections'
			echo '# see https://www.postgresql.org/docs/12/auth-trust.html'
		fi
		echo "host all all all $POSTGRES_HOST_AUTH_METHOD"
	} >> "$PGDATA/pg_hba.conf"
}

# start socket-only postgresql server for setting up or running scripts
# all arguments will be passed along as arguments to `postgres` (via pg_ctl)
docker_temp_server_start() {
	if [ "$1" = 'postgres' ]; then
		shift
	fi

	# internal start of server in order to allow setup using psql client
	# does not listen on external TCP/IP and waits until start finishes
	set -- "$@" -c listen_addresses='' -p "${PGPORT:-5432}"

	PGUSER="${PGUSER:-$POSTGRES_USER}" \
	pg_ctl -D "$PGDATA" \
		-o "$(printf '%q ' "$@")" \
		-w start
}

# stop postgresql server after done setting up user and running scripts
docker_temp_server_stop() {
	PGUSER="${PGUSER:-postgres}" \
	pg_ctl -D "$PGDATA" -m fast -w stop
}

# check arguments for an option that would cause postgres to stop
# return true if there is one
_pg_want_help() {
	local arg
	for arg; do
		case "$arg" in
			# postgres --help | grep 'then exit'
			# leaving out -C on purpose since it always fails and is unhelpful:
			# postgres: could not access the server configuration file "/var/lib/postgresql/data/postgresql.conf": No such file or directory
			-'?'|--help|--describe-config|-V|--version)
				return 0
				;;
		esac
	done
	return 1
}

_main() {
	# if first arg looks like a flag, assume we want to run postgres server
	if [ "${1:0:1}" = '-' ]; then
		set -- postgres "$@"
	fi

	if [ "$1" = 'postgres' ] && ! _pg_want_help "$@"; then
		docker_setup_env
		# setup data directories and permissions (when run as root)
		docker_create_db_directories
		if [ "$(id -u)" = '0' ]; then
			# then restart script as postgres user
			exec gosu postgres "$BASH_SOURCE" "$@"
		fi

		# only run initialization on an empty data directory
		if [ -z "$DATABASE_ALREADY_EXISTS" ]; then
			docker_verify_minimum_env

			# check dir permissions to reduce likelihood of half-initialized database
			ls /docker-entrypoint-initdb.d/ > /dev/null

			docker_init_database_dir
			pg_setup_hba_conf "$@"

			# PGPASSWORD is required for psql when authentication is required for 'local' connections via pg_hba.conf and is otherwise harmless
			# e.g. when '--auth=md5' or '--auth-local=md5' is used in POSTGRES_INITDB_ARGS
			export PGPASSWORD="${PGPASSWORD:-$POSTGRES_PASSWORD}"
			docker_temp_server_start "$@"

			docker_setup_db
			docker_process_init_files /docker-entrypoint-initdb.d/*

			docker_temp_server_stop
			unset PGPASSWORD

			echo
			echo 'PostgreSQL init process complete; ready for start up.'
			echo
		else
			echo
			echo 'PostgreSQL Database directory appears to contain a database; Check for upgrade'
			docker_process_init_files /usr/share/container-scripts/postgresql/*
            # pulled from slc image common.sh
            local versionfile="$PGDATA"/PG_VERSION version upgrade_available

            # This file always exists.
            test -f "$versionfile"
            version=$(cat "$versionfile")

            if test "13" = "$version"; then
                # try upgrade
                echo "Found database of version $version time to try to upgrade"
                export POSTGRESQL_PREV_VERSION="13"
                export POSTGRESQL_VERSION="15"
                export POSTGRESQL_UPGRADE="copy"
                export POSTGRESQL_LOG_DESTINATION="/dev/stderr"
#                run_pgupgrade
                echo 'Running copy of upgrade to see where it falls down'
                shrews_pgupgrade
                echo 'Done running copy of upgrade to see where it falls down'
            fi

#
#			cat "$PGDATA/postgresql.conf"
#			export POSTGRESQL_LOG_DESTINATION=/dev/stderr
#			cat /usr/share/container-scripts/postgresql/common.sh
#			try_pgupgrade
			echo
		fi
	fi

	flock /var/lib/postgresql/data/pglock "$@" &
	child=$!
	echo "Waiting for child process $child to exit"
	wait "$child"
}

shrews_pgupgrade ()
(
  # Remove .pid file if the file persists after ugly shut down
  if [ -f "$PGDATA/postmaster.pid" ] && ! pg_isready > /dev/null; then
    rm -rf "$PGDATA/postmaster.pid"
  fi

  optimized=false
  old_raw_version=${POSTGRESQL_PREV_VERSION//\./}
  new_raw_version=${POSTGRESQL_VERSION//\./}

  old_pgengine=/usr/lib64/pgsql/postgresql-$old_raw_version/bin
  new_pgengine=/usr/bin

  PGDATA_new="${PGDATA}-new"

  printf >&2 "\n==========  \$PGDATA upgrade: %s -> %s  ==========\n\n" \
             "$POSTGRESQL_PREV_VERSION" \
             "$POSTGRESQL_VERSION"

  info_msg () { printf >&2 "\n===>  $*\n\n" ;}

  # pg_upgrade writes logs to cwd, so go to the persistent storage first
  cd "$HOME"/data
  echo "$pwd"

  # disable this because of scl_source, 'set +u' just makes the code ugly
  # anyways
  set +u

  case $POSTGRESQL_UPGRADE in
    copy) # we accept this
      ;;
    hardlink)
      optimized=:
      ;;
    *)
      echo >&2 "Unsupported value: \$POSTGRESQL_UPGRADE=$POSTGRESQL_UPGRADE"
      false
      ;;
  esac

  # boot up data directory with old postgres once again to make sure
  # it was shut down properly, otherwise the upgrade process fails
  info_msg "Starting old postgresql once again for a clean shutdown..."
  "${old_pgengine}/pg_ctl" start -w --timeout 86400 -o "-h 127.0.0.1''"
  info_msg "Waiting for postgresql to be ready for shutdown again..."
  "${old_pgengine}/pg_isready" -h 127.0.0.1
  info_msg "Shutting down old postgresql cleanly..."
  "${old_pgengine}/pg_ctl" stop
  info_msg "SHREWS -0- Shut down old, about to test PGDATA_new"

  # Ensure $PGDATA_new doesn't exist yet, so we can immediately remove it if
  # there's some problem.
#  test ! -e "$PGDATA_new"

  # initialize the database
  info_msg "Initialize new data directory; we will migrate to that."
  initdb_cmd=( initdb_wrapper "$new_pgengine"/initdb "$PGDATA_new" )
  eval "\${initdb_cmd[@]} ${POSTGRESQL_UPGRADE_INITDB_OPTIONS-}" || \
    { rm -rf "$PGDATA_new" ; false ; }

  info_msg "building upgrade command"
  upgrade_cmd=(
      "$new_pgengine"/pg_upgrade
      "--old-bindir=$old_pgengine"
      "--new-bindir=$new_pgengine"
      "--old-datadir=$PGDATA"
      "--new-datadir=$PGDATA_new"
  )

  # Dangerous --link option, we loose $DATADIR if something goes wrong.
  ! $optimized || upgrade_cmd+=(--link)

  # User-specififed options for pg_upgrade.
  eval "upgrade_cmd+=(${POSTGRESQL_UPGRADE_PGUPGRADE_OPTIONS-})"

  # On non-intel arches the data_sync_retry set to on
  sed -i -e 's/data_sync_retry/#data_sync_retry/' "${POSTGRESQL_CONFIG_FILE}"

  # the upgrade
  info_msg "Starting the pg_upgrade process."

  # Once we stop support for PostgreSQL 9.4, we don't need
  # REDHAT_PGUPGRADE_FROM_RHEL hack as we don't upgrade from 9.2 -- that means
  # that we don't need to fiddle with unix_socket_director{y,ies} option.
  REDHAT_PGUPGRADE_FROM_RHEL=1 \
  "${upgrade_cmd[@]}" || { cat $(find "$PGDATA_new"/.. -name pg_upgrade_server.log) ; rm -rf "$PGDATA_new" && false ; }

  # Move the important configuration and remove old data.  This is highly
  # careless, but we can't do more for this over-automatized process.
  info_msg "Swap the old and new PGDATA and cleanup."
  mv "$PGDATA"/*.conf "$PGDATA_new"
  rm -rf "$PGDATA"
  mv "$PGDATA_new" "$PGDATA"

  # Get back the option we changed above
  sed -i -e 's/#data_sync_retry/data_sync_retry/' "${POSTGRESQL_CONFIG_FILE}"

  info_msg "Upgrade DONE."
)

if ! _is_sourced; then
	_main "$@"
fi
