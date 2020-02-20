-- +goose Up
-- SQL in this section is executed when the migration is applied.
 
CREATE TABLE IF NOT EXISTS `batch_requests` (
        `id` VARCHAR(36),
        `denomination_id` VARCHAR(36)  NOT NULL,
        `status` VARCHAR(100)  NOT NULL DEFAULT 'PENDING',
        `created_at` DATETIME NULL,
        `updated_at` DATETIME NULL,
        `date_completed` DATETIME NULL,
        `date_of_processing` DATETIME NULL,
        `records` int,

        PRIMARY KEY (`id`),
        INDEX status (status), 
        INDEX denomination_id (denomination_id)
);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS batch_requests;
