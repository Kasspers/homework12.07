CREATE TABLE IF NOT EXISTS user_roles (
                                    id serial PRIMARY KEY,
                                    user_id INT,
                                    role_id INT,
                                    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
                                    FOREIGN KEY (role_id) REFERENCES roles (id)
);


insert into user_roles (user_id,role_id) values (1,2);
insert into user_roles (user_id,role_id) values (2,1);
insert into user_roles (user_id,role_id) values (3,1);
insert into user_roles (user_id,role_id) values (4,1);