-- Rollback initial schema for Kasaneha AI diary application

-- Drop sequence
DROP SEQUENCE IF EXISTS message_sequence_seq;

-- Drop triggers
DROP TRIGGER IF EXISTS update_chat_sessions_updated_at ON chat_sessions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_active_sessions;
DROP INDEX IF EXISTS idx_sessions_user_month;
DROP INDEX IF EXISTS idx_user_statistics_user_id;
DROP INDEX IF EXISTS idx_analyses_keywords;
DROP INDEX IF EXISTS idx_analyses_emotional_state;
DROP INDEX IF EXISTS idx_analyses_created_at;
DROP INDEX IF EXISTS idx_analyses_tension_score;
DROP INDEX IF EXISTS idx_analyses_session_id;
DROP INDEX IF EXISTS idx_messages_sender;
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_session_sequence;
DROP INDEX IF EXISTS idx_messages_session_id;
DROP INDEX IF EXISTS idx_chat_sessions_status;
DROP INDEX IF EXISTS idx_chat_sessions_user_date;
DROP INDEX IF EXISTS idx_chat_sessions_date;
DROP INDEX IF EXISTS idx_chat_sessions_user_id;
DROP INDEX IF EXISTS idx_users_active;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS user_statistics;
DROP TABLE IF EXISTS analyses;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS chat_sessions;
DROP TABLE IF EXISTS users;

-- Drop extensions (optional, commented out as they might be used by other applications)
-- DROP EXTENSION IF EXISTS "pgcrypto";
-- DROP EXTENSION IF EXISTS "uuid-ossp"; 