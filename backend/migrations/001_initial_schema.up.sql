-- Initial schema for Kasaneha AI diary application
-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Constraints
    CONSTRAINT users_username_check CHECK (length(username) >= 3),
    CONSTRAINT users_email_check CHECK (email ~* '^[^@]+@[^@]+\.[^@]+$')
);

-- Chat sessions table
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- One session per user per day
    UNIQUE(user_id, session_date)
);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(10) NOT NULL CHECK (sender IN ('user', 'ai')),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    sequence_number INTEGER NOT NULL
);

-- Analyses table
CREATE TABLE analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    emotional_state JSONB NOT NULL,
    behavioral_insights JSONB NOT NULL,
    tension_score INTEGER NOT NULL CHECK (tension_score >= 0 AND tension_score <= 100),
    relative_score INTEGER CHECK (relative_score >= -50 AND relative_score <= 50),
    keywords JSONB DEFAULT '[]'::jsonb,
    raw_analysis_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- One analysis per session
    UNIQUE(session_id)
);

-- User statistics table (cached data)
CREATE TABLE user_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_sessions INTEGER DEFAULT 0,
    average_tension_score DECIMAL(5,2),
    min_tension_score INTEGER,
    max_tension_score INTEGER,
    most_common_emotions JSONB DEFAULT '[]'::jsonb,
    last_calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- One statistics record per user
    UNIQUE(user_id)
);

-- Indexes for performance

-- Users indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Chat sessions indexes
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_date ON chat_sessions(session_date);
CREATE INDEX idx_chat_sessions_user_date ON chat_sessions(user_id, session_date);
CREATE INDEX idx_chat_sessions_status ON chat_sessions(status);

-- Messages indexes
CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_session_sequence ON messages(session_id, sequence_number);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_sender ON messages(sender);

-- Analyses indexes
CREATE INDEX idx_analyses_session_id ON analyses(session_id);
CREATE INDEX idx_analyses_tension_score ON analyses(tension_score);
CREATE INDEX idx_analyses_created_at ON analyses(created_at);

-- JSONB indexes for searching
CREATE INDEX idx_analyses_emotional_state ON analyses USING GIN (emotional_state);
CREATE INDEX idx_analyses_keywords ON analyses USING GIN (keywords);

-- User statistics indexes
CREATE INDEX idx_user_statistics_user_id ON user_statistics(user_id);

-- Composite indexes for common queries
CREATE INDEX idx_sessions_user_month ON chat_sessions(user_id, EXTRACT(YEAR FROM session_date), EXTRACT(MONTH FROM session_date));

-- Partial indexes for active sessions
CREATE INDEX idx_active_sessions ON chat_sessions(user_id, session_date) WHERE status = 'active';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sequence for message ordering
CREATE SEQUENCE message_sequence_seq; 