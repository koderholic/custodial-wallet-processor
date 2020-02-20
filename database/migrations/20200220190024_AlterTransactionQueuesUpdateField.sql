-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE transaction_queues MODIFY value BIGINT(20);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
ALTER TABLE transaction_queues MODIFY value DECIMAL(64,18);