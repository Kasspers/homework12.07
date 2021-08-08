CREATE TABLE IF NOT EXISTS users (
                                     id serial PRIMARY KEY,
                                     name VARCHAR ( 20 ) UNIQUE NOT NULL,
                                     password VARCHAR ( 20 ) UNIQUE NOT NULL
);

insert into users (name, password) values ('admin', 'admin');
insert into users (name, password) values ('user', 'user');
insert into users (name, password) values ('random', 'dave');