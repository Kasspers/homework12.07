CREATE TABLE IF NOT EXISTS rental_history (
                                    rental_id serial PRIMARY KEY,
                                    book_id INT NOT NULL,
                                    rental_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                    return_date TIMESTAMP,
                                    reader_id INT NOT NULL,
                                    FOREIGN KEY (book_id) REFERENCES book (book_id),
                                    FOREIGN KEY (reader_id) REFERENCES reader (reader_id)
);