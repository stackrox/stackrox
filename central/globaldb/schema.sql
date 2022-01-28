CREATE OR REPLACE FUNCTION random_between(low INT ,high INT)
    RETURNS INT AS
$$
BEGIN
RETURN floor(random()* (high-low + 1) + low);
END;
$$ language 'plpgsql' STRICT;

SELECT random_between(1,100)
FROM generate_series(1,5);


DROP TABLE IF EXISTS c;
CREATE TABLE c (
                   id SERIAL UNIQUE NOT NULL PRIMARY KEY,
                   name VARCHAR NOT NULL
);
insert into c (name)
select md5(random()::text)
from generate_series(1, 10000) s(i);

DROP TABLE IF EXISTS n;
CREATE TABLE n (
                   id SERIAL UNIQUE NOT NULL PRIMARY KEY,
                   name VARCHAR NOT NULL
);
insert into n (name)
select md5(random()::text)
from generate_series(1, 10000) s(i);

DROP TABLE IF EXISTS test;
CREATE TABLE test (
                      id SERIAL UNIQUE NOT NULL PRIMARY KEY,
                      c_id INT NOT NULL REFERENCES c,
                      n_id INT NOT NULL REFERENCES n
);

insert into test (
    c_id, n_id
)
select
    random_between(1,1000),
    random_between(1,1000)
from generate_series(1, 10000000) s(i);

DROP INDEX c_idx;
CREATE INDEX c_idx ON test USING hash (c_id);
DROP INDEX n_idx;
CREATE INDEX n_idx ON test USING hash (n_id);

SELECT
    indexname,
    indexdef
FROM
    pg_indexes
WHERE
        tablename = 'test';

select * from generate_series(0, 1001);

select count(*) from test;
explain select count(*) from test where c_id = 0;
explain select count(*) from test where
    (c_id = 1 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 2 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 3 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 4 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 5 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 6 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 7 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 8 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 9 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 10 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 11 and n_id IN (1,2,3,4,5,6,7,8,9,10)) OR
    (c_id = 12 and n_id IN (1,2,3,4,5,6,7,8,9,10))
;

explain select count(*) from test where (c_id = 1 and n_id BETWEEN 1 AND 1000);
