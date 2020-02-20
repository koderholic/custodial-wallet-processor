-- +goose Up
-- SQL in this section is executed when the migration is applied.

CREATE TABLE IF NOT EXISTS `user_addresses` (
    `id` varchar(36) NOT NULL, 
    `created_at` timestamp NULL, 
    `updated_at` timestamp NULL, 
    `asset_id` varchar(36) NOT NULL, 
    `address` varchar(255) NOT NULL, 
    `is_valid` tinyint(1) DEFAULT 1, 
    
    PRIMARY KEY (id), 
    CONSTRAINT user_addresses_asset_id_asset_id_foreign FOREIGN KEY (asset_id) REFERENCES `user_assets` (`id`) ON DELETE CASCADE ON UPDATE CASCADE, 
    INDEX asset_id (asset_id)
);


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS user_addresses;