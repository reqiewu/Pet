
CREATE TYPE pet_status AS ENUM ('available', 'pending', 'sold');

CREATE TYPE order_status AS ENUM ('placed', 'approved', 'delivered');

CREATE TABLE users (
                       id SERIAL PRIMARY KEY,
                       username VARCHAR(255) UNIQUE NOT NULL,
                       first_name VARCHAR(255),
                       last_name VARCHAR(255),
                       email VARCHAR(255) UNIQUE,
                       password_hash VARCHAR(255) NOT NULL,
                       phone VARCHAR(50),
                       user_status INTEGER DEFAULT 1,
                       token TEXT
);


CREATE TABLE pet_categories (
                                id SERIAL PRIMARY KEY,
                                name VARCHAR(255) NOT NULL
);


CREATE TABLE pet_tags (
                          id SERIAL PRIMARY KEY,
                          name VARCHAR(255) NOT NULL
);

-
CREATE TABLE pets (
                      id SERIAL PRIMARY KEY,
                      name VARCHAR(255) NOT NULL,
                      category_id INTEGER REFERENCES pet_categories(id),
                      status pet_status NOT NULL,
                      photo_urls TEXT[] DEFAULT '{}',
                      owner_id INTEGER REFERENCES users(id)
);


CREATE TABLE pet_to_tags (
                             pet_id INTEGER REFERENCES pets(id) ON DELETE CASCADE,
                             tag_id INTEGER REFERENCES pet_tags(id) ON DELETE CASCADE,
                             PRIMARY KEY (pet_id, tag_id)
);

-- Таблица заказов
CREATE TABLE orders (
                        id SERIAL PRIMARY KEY,
                        pet_id INTEGER REFERENCES pets(id),
                        user_id INTEGER REFERENCES users(id),
                        quantity INTEGER NOT NULL,
                        ship_date TIMESTAMP,
                        status order_status NOT NULL,
                        complete BOOLEAN DEFAULT FALSE
);