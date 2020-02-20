-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE IF NOT EXISTS `user_assets` (
    `id` varchar(36) NOT NULL, 
    `created_at` timestamp NULL, 
    `updated_at` timestamp NULL, 
    `user_id` varchar(36) NOT NULL, 
    `denomination_id` varchar(36) NOT NULL, 
    `available_balance` decimal(64,18) NOT NULL, 
    `deleted_at` timestamp NULL, PRIMARY KEY (id), 
    
    CONSTRAINT user_assets_denomination_id_denominations_id_foreign FOREIGN KEY (denomination_id) REFERENCES `denominations` (`id`) ON DELETE NO ACTION ON UPDATE NO ACTION, 
    INDEX user_id (user_id), 
    INDEX denomination_id (denomination_id), 
    INDEX idx_user_assets_deleted_at (deleted_at)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

DROP TABLE IF EXISTS user_assets;