# ACS Database Schema Guide

## Table Of Contents
- [Background](#background)
- [Code Generation](#code-generation)
    - [Schema Definitions](#schema-definitions)
- [Guidelines](#guidelines)

## Background
ACS is backed by a Postgres database.  ACS uses various code generation techniques to generate:
- Postgres Schema
- Stores to provide an interface to the underlying database

This allows ACS to restrict the database features that are supported as well as provide a consistent
means for creating and interacting with the database.  The individual Postgres tables only contain
fields necessary for search and a `Serialized` byte array field.  The data of truth is contained
within the `Serialized` field.  The other columns are only used for the purposes of searching or aggregation 
functions.

## Code Generation
The source of the schema definitions is contained with in the proto files contained in `./proto/storage`.  `Go` tags
are used to define:
- Fields stored as searchable columns
- Fields with indexes
- Fields with constraints (i.e. unique constraints)
- Fields with foreign keys references to other tables
- Fields that reference fields in other tables but no foreign key is present but the search framework will
    generate the logical relationship



### Schema Definitions

## Guidelines