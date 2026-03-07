
CREATE USER morrigan WITH LOGIN PASSWORD 'Develop2023';
CREATE DATABASE morrigan WITH OWNER = morrigan ENCODING = 'UTF8';
GRANT ALL PRIVILEGES ON DATABASE morrigan to morrigan;


\c morrigan

ALTER SCHEMA public OWNER TO morrigan;

CREATE EXTENSION vector;



-- testing
CREATE DATABASE morrigan_test WITH OWNER = morrigan ENCODING = 'UTF8';
\c morrigan_test
