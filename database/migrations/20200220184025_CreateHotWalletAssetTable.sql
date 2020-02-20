-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS `hot_wallet_assets` (
    `id` varchar(36) NOT NULL, 
    `created_at` timestamp NULL, 
    `updated_at` timestamp NULL, 
    `address` varchar(255) NOT NULL, 
    `asset_symbol` varchar(255) NOT NULL, 
    `balance` bigint, 
    `is_disabled` tinyint(1) DEFAULT 1, 

    PRIMARY KEY (id), 
    CONSTRAINT uix_hot_wallet_assets_asset_symbol UNIQUE (asset_symbol)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS hot_wallet_assets;

