CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    from_account_id INT NOT NULL,
    to_account_id INT NOT NULL,
    amount NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_from_account
        FOREIGN KEY(from_account_id) 
        REFERENCES accounts(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_to_account
        FOREIGN KEY(to_account_id) 
        REFERENCES accounts(id)
        ON DELETE CASCADE
);