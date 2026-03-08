-- 初始化评论分表与点赞日志表。

DELIMITER $$

DROP PROCEDURE IF EXISTS create_comment_shards$$

CREATE PROCEDURE create_comment_shards()
BEGIN
    DECLARE i INT DEFAULT 0;
    DECLARE table_name VARCHAR(64);
    DECLARE ddl_stmt LONGTEXT;

    WHILE i < 8 DO
        SET table_name = CONCAT('comment_shard_', i);
        SET ddl_stmt = CONCAT(
            'CREATE TABLE IF NOT EXISTS `', table_name, '` (',
            '  `id` BIGINT AUTO_INCREMENT PRIMARY KEY,',
            '  `comment_id` VARCHAR(64) NOT NULL,',
            '  `post_id` VARCHAR(64) NOT NULL,',
            '  `parent_id` VARCHAR(64) DEFAULT NULL,',
            '  `author` VARCHAR(128) NOT NULL,',
            '  `author_ip` VARCHAR(45) DEFAULT NULL,',
            '  `content` TEXT NOT NULL,',
            '  `status` TINYINT NOT NULL DEFAULT 0,',
            '  `like_count` INT NOT NULL DEFAULT 0,',
            '  `is_hot` TINYINT NOT NULL DEFAULT 0,',
            '  `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),',
            '  `updated_at` DATETIME(3) ON UPDATE CURRENT_TIMESTAMP(3),',
            '  UNIQUE KEY `uk_comment_id` (`comment_id`),',
            '  KEY `idx_post_created` (`post_id`, `created_at` DESC, `id` DESC),',
            '  KEY `idx_post_status` (`post_id`, `status`),',
            '  KEY `idx_author` (`author`),',
            '  KEY `idx_created_at` (`created_at`)',
            ') ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci'
        );

        PREPARE stmt FROM ddl_stmt;
        EXECUTE stmt;
        DEALLOCATE PREPARE stmt;

        SET i = i + 1;
    END WHILE;
END$$

DELIMITER ;

CALL create_comment_shards();
DROP PROCEDURE IF EXISTS create_comment_shards;

CREATE TABLE IF NOT EXISTS `comment_like_log` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `comment_id` VARCHAR(64) NOT NULL,
    `user_identifier` VARCHAR(128) NOT NULL,
    `created_at` DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY `uk_comment_user` (`comment_id`, `user_identifier`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
