-- XNAuth 初始数据库结构
-- 用途：新库初始化。旧库升级请先备份后按实际差异处理。

CREATE DATABASE IF NOT EXISTS `auth` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE `auth`;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS `apps` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_key` VARCHAR(64) NOT NULL,
  `app_name` VARCHAR(128) NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `min_login_version_code` INT DEFAULT NULL,
  `force_update` TINYINT NOT NULL DEFAULT 0,
  `secure_key_id` VARCHAR(128) DEFAULT NULL,
  `secure_x25519_private_key` VARCHAR(128) DEFAULT NULL,
  `secure_x25519_public_key` VARCHAR(128) DEFAULT NULL,
  `secure_ed25519_private_key` VARCHAR(128) DEFAULT NULL,
  `secure_ed25519_public_key` VARCHAR(128) DEFAULT NULL,
  `remark` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_key` (`app_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `admin_users` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `username` VARCHAR(64) NOT NULL,
  `password_hash` VARCHAR(255) NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `last_login_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `system_settings` (
  `setting_key` VARCHAR(128) NOT NULL,
  `setting_value` TEXT NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`setting_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `license_cards` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `license_key` VARCHAR(128) NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 0,
  `remark` VARCHAR(255) DEFAULT NULL,
  `max_devices` INT NOT NULL DEFAULT 1,
  `max_online` INT NOT NULL DEFAULT 1,
  `expire_at` DATETIME DEFAULT NULL,
  `activated_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_license` (`app_id`, `license_key`),
  KEY `idx_app_status` (`app_id`, `status`),
  KEY `idx_expire_at` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `license_devices` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `license_id` BIGINT NOT NULL,
  `machine_code_hash` VARCHAR(128) NOT NULL,
  `device_name` VARCHAR(128) DEFAULT NULL,
  `device_public_key` VARCHAR(128) DEFAULT NULL,
  `device_key_bound_at` DATETIME DEFAULT NULL,
  `client_version` VARCHAR(64) DEFAULT NULL,
  `client_version_code` INT DEFAULT NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `first_seen_at` DATETIME NOT NULL,
  `last_seen_at` DATETIME NOT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_license_machine` (`license_id`, `machine_code_hash`),
  KEY `idx_app_license` (`app_id`, `license_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `client_nonces` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `device_id` BIGINT NOT NULL,
  `nonce` VARCHAR(128) NOT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_device_nonce` (`device_id`, `nonce`),
  KEY `idx_app_device_time` (`app_id`, `device_id`, `created_at`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `license_sessions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `license_id` BIGINT NOT NULL,
  `device_id` BIGINT NOT NULL,
  `session_token` VARCHAR(128) NOT NULL,
  `status` TINYINT NOT NULL DEFAULT 1,
  `client_ip` VARCHAR(64) DEFAULT NULL,
  `client_version` VARCHAR(64) DEFAULT NULL,
  `client_version_code` INT DEFAULT NULL,
  `started_at` DATETIME NOT NULL,
  `last_heartbeat_at` DATETIME NOT NULL,
  `revoked_at` DATETIME DEFAULT NULL,
  `revoke_reason` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_token` (`session_token`),
  KEY `idx_license_status` (`license_id`, `status`),
  KEY `idx_device_status` (`device_id`, `status`),
  KEY `idx_heartbeat` (`last_heartbeat_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `app_announcements` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `title` VARCHAR(128) NOT NULL,
  `content` TEXT NOT NULL,
  `notice_type` VARCHAR(32) DEFAULT 'normal',
  `popup` TINYINT NOT NULL DEFAULT 0,
  `enabled` TINYINT NOT NULL DEFAULT 1,
  `start_at` DATETIME DEFAULT NULL,
  `end_at` DATETIME DEFAULT NULL,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_enabled` (`app_id`, `enabled`),
  KEY `idx_time_range` (`start_at`, `end_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `app_versions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `version_name` VARCHAR(64) NOT NULL,
  `version_code` INT NOT NULL,
  `min_supported_code` INT DEFAULT NULL,
  `download_url` VARCHAR(512) DEFAULT NULL,
  `file_hash` VARCHAR(128) DEFAULT NULL,
  `file_size` BIGINT DEFAULT NULL,
  `changelog` TEXT,
  `force_update` TINYINT NOT NULL DEFAULT 0,
  `enabled` TINYINT NOT NULL DEFAULT 1,
  `released_at` DATETIME DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_app_version` (`app_id`, `version_code`),
  KEY `idx_enabled` (`enabled`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `collect_fields` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `field_key` VARCHAR(128) NOT NULL,
  `field_name` VARCHAR(128) NOT NULL,
  `enabled` TINYINT NOT NULL DEFAULT 1,
  `show_in_list` TINYINT NOT NULL DEFAULT 0,
  `stat_enabled` TINYINT NOT NULL DEFAULT 0,
  `stat_type` VARCHAR(20) NOT NULL DEFAULT 'distribution',
  `search_enabled` TINYINT NOT NULL DEFAULT 0,
  `sort_order` INT NOT NULL DEFAULT 0,
  `created_at` DATETIME NOT NULL,
  `updated_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_app_field` (`app_id`, `field_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `collect_records` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `license_id` BIGINT DEFAULT NULL,
  `device_id` BIGINT DEFAULT NULL,
  `license_key` VARCHAR(128) NOT NULL,
  `machine_code_hash` VARCHAR(128) DEFAULT NULL,
  `event` VARCHAR(64) DEFAULT 'custom',
  `client_ip` VARCHAR(64) DEFAULT NULL,
  `user_agent` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_license_time` (`license_id`, `created_at`),
  KEY `idx_device_time` (`device_id`, `created_at`),
  KEY `idx_event_time` (`app_id`, `event`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `collect_record_values` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `record_id` BIGINT NOT NULL,
  `app_id` BIGINT NOT NULL,
  `license_id` BIGINT DEFAULT NULL,
  `device_id` BIGINT DEFAULT NULL,
  `field_key` VARCHAR(128) NOT NULL,
  `field_value` VARCHAR(2048) NOT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_record` (`record_id`),
  KEY `idx_field_time` (`app_id`, `field_key`, `created_at`),
  KEY `idx_field_value` (`app_id`, `field_key`, `field_value`(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `verify_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `app_id` BIGINT NOT NULL,
  `license_id` BIGINT DEFAULT NULL,
  `device_id` BIGINT DEFAULT NULL,
  `license_key` VARCHAR(128) DEFAULT NULL,
  `machine_code_hash` VARCHAR(128) DEFAULT NULL,
  `client_ip` VARCHAR(64) DEFAULT NULL,
  `client_version` VARCHAR(64) DEFAULT NULL,
  `client_version_code` INT DEFAULT NULL,
  `result` TINYINT NOT NULL,
  `fail_reason` VARCHAR(255) DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_license_time` (`license_id`, `created_at`),
  KEY `idx_result_time` (`app_id`, `result`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `admin_id` BIGINT DEFAULT NULL,
  `module` VARCHAR(64) NOT NULL,
  `action` VARCHAR(64) NOT NULL,
  `target_id` BIGINT DEFAULT NULL,
  `content` TEXT,
  `client_ip` VARCHAR(64) DEFAULT NULL,
  `created_at` DATETIME NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_admin_time` (`admin_id`, `created_at`),
  KEY `idx_module_time` (`module`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `system_settings` (`setting_key`, `setting_value`, `created_at`, `updated_at`) VALUES
  ('login_captcha_enabled', '0', NOW(), NOW()),
  ('site_name', 'XNAuth 汐念验证', NOW(), NOW()),
  ('icp_number', '', NOW(), NOW()),
  ('footer_links', '[{"label":"进入后台","url":"/login"},{"label":"接口健康","url":"/api/health"}]', NOW(), NOW())
ON DUPLICATE KEY UPDATE `setting_key` = VALUES(`setting_key`);

SET FOREIGN_KEY_CHECKS = 1;
