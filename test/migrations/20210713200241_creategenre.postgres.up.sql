CREATE TABLE IF NOT EXISTS genre (
                                      genre_id serial PRIMARY KEY,
                                      genre VARCHAR ( 50 ) UNIQUE NOT NULL
);