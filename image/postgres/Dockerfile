FROM postgres:14

RUN  sed -i "s/#shared_preload_libraries = ''/shared_preload_libraries = 'pg_stat_statements'/g" /usr/share/postgresql/14/postgresql.conf.sample
