CREATE TABLE IF NOT EXISTS author (
                                      author_id serial PRIMARY KEY,
                                      author_name VARCHAR ( 50 ) UNIQUE NOT NULL
);