CREATE TABLE IF NOT EXISTS reader (
                                     reader_id serial PRIMARY KEY,
                                     name VARCHAR ( 50 ) UNIQUE NOT NULL,
                                     birth_date DATE NOT NULL,
                                     registration_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);