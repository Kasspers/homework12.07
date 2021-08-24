CREATE TABLE IF NOT EXISTS book_load_tokens (
                                        id serial PRIMARY KEY,
                                        token VARCHAR (255) NOT NULL,
                                        book_id int NOT NULL,
                                        created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                        FOREIGN KEY (book_id) REFERENCES book (book_id) ON DELETE CASCADE
);