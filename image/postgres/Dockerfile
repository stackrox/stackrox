FROM postgres:14

RUN  sed -i "s/#shared_preload_libraries = ''/shared_preload_libraries = 'pg_stat_statements'/g" /usr/share/postgresql/14/postgresql.conf.sample
RUN sed -i "s/#max_wal_size = 1GB/max_wal_size = 5GB/g" /usr/share/postgresql/14/postgresql.conf.sample
