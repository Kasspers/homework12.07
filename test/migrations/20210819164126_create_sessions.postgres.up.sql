CREATE TABLE IF NOT EXISTS sessions (
                                        user_id INT not null,
                                         refresh_token VARCHAR (255) not null,
                                        session_start TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                                         FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);