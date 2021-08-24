CREATE TABLE IF NOT EXISTS book (
                                      book_id serial PRIMARY KEY,
                                      name VARCHAR ( 50 ) UNIQUE NOT NULL,
                                      release_date DATE NOT NULL,
                                      author_id INT NOT NULL,
                                      current_reader INT,
                                      genre_id INT,
                                      book_filepath varchar(255),
                                      image_filepath varchar(255),
                                      FOREIGN KEY (author_id) REFERENCES author (author_id),
                                    FOREIGN KEY (current_reader) REFERENCES reader (reader_id),
                                    FOREIGN KEY (genre_id) REFERENCES genre (genre_id)
);