-- Create "chats" table
CREATE TABLE `chats` (
  `id` char(36) NOT NULL,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `deleted_at` datetime(3) NULL,
  `object` longtext NULL,
  `title` longtext NULL,
  `session_id` varchar(191) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_chats_created_at` (`created_at`),
  INDEX `idx_chats_deleted_at` (`deleted_at`),
  INDEX `idx_chats_session_id` (`session_id`),
  INDEX `idx_chats_updated_at` (`updated_at`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
-- Create "qas" table
CREATE TABLE `qas` (
  `id` char(36) NOT NULL,
  `chat_id` char(36) NULL,
  `session_id` varchar(36) NULL,
  `job_group` varchar(36) NULL,
  `question` text NULL,
  `answer` text NULL,
  `references` json NULL,
  `keywords` json NULL,
  `related_queries` json NULL,
  `vote` varchar(191) NULL,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `status` longtext NULL,
  `token_count` bigint NULL,
  `llm_provider` varchar(191) NULL,
  `llm_model` varchar(191) NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_qas_chat_id` (`chat_id`),
  INDEX `idx_qas_created_at` (`created_at`),
  INDEX `idx_qas_job_group` (`job_group`),
  INDEX `idx_qas_llm_model` (`llm_model`),
  INDEX `idx_qas_llm_provider` (`llm_provider`),
  FULLTEXT INDEX `idx_qas_question` (`question`) WITH PARSER `ngram`,
  INDEX `idx_qas_session_id` (`session_id`),
  INDEX `idx_qas_vote` (`vote`)
) CHARSET utf8mb4 COLLATE utf8mb4_general_ci;