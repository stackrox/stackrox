# Frozen Schemas
The schemas in Central are evolving. Migrations are created to upgrade Central
databases from one version to another. Data migrations are based on the schemas at
that particular release and should not evolve with Central.

We create a copy of schemas and freeze it here so that the migrations can use them to upgrade
legacy databases. All schemas are frozen for version
3.73 to support upgrade from RocksDB and BoltDb to Postgres. Note this is not mandatory for every release. In a more common scenario,
if we need to create one migration for one table, you can keep the frozen schema in the migration itself.
