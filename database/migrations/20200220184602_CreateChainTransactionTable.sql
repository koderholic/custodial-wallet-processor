-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS `chain_transactions` (
    `id` varchar(36) NOT NULL, 
    `created_at` timestamp NULL, 
    `updated_at` timestamp NULL, 
    `status` tinyint(1) DEFAULT 0 NOT NULL, 
    `batch_id` varchar(36), 
    `transaction_hash` varchar(255) NOT NULL, 
    `block_height` bigint, 
    `transaction_fee` varchar(255),

    PRIMARY KEY (id), 
    INDEX idx_chain_transactions_status (status), 
    INDEX batch_id (batch_id)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS chain_transactions;
