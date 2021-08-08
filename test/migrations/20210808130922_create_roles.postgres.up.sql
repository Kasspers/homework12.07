CREATE TABLE IF NOT EXISTS roles (
                                     id serial PRIMARY KEY,
                                     role VARCHAR ( 20 ) UNIQUE NOT NULL
);

insert into roles (role) values ('reader');
insert into roles (role) values ('librarian');