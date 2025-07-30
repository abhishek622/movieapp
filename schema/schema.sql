CREATE TABLE IF NOT EXISTS movies (
    id VARCHAR(255) primary KEY,
    title VARCHAR(255),
    director VARCHAR(255),
    description TEXT
);

CREATE TABLE IF NOT EXISTS ratings (
    record_id VARCHAR(255),
    record_type VARCHAR(255),
    user_id VARCHAR(255),
    value INT,
    PRIMARY KEY (record_id, record_type, user_id)
);