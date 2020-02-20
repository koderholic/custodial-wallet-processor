-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS `denominations` (
    `id` varchar(36) NOT NULL, 
    `created_at` timestamp NULL, 
    `updated_at` timestamp NULL, 
    `name` varchar(255), 
    `symbol` varchar(255) NOT NULL, 
    `token_type` varchar(255) NOT NULL, 
    `decimal` int, 
    `is_enabled` tinyint(1) DEFAULT 1, 

    PRIMARY KEY (id), 
    CONSTRAINT uix_denominations_symbol UNIQUE (symbol), 
    INDEX isEnabled (is_enabled),
    INDEX symbol (symbol)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS denominations;


