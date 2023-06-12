BEGIN;

CREATE TABLE IF NOT EXISTS logs (
                                    id SERIAL PRIMARY KEY,
                                    unix_ts BIGINT NOT NULL,
                                    user_id INT NOT NULL,
                                    event_name VARCHAR(100) NOT NULL
    );

COMMIT;