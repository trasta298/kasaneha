// User types
export interface User {
  id: string;
  username: string;
  email?: string;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  is_active: boolean;
  timezone: string;
}

// Authentication types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  password: string;
  email?: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

// Chat Session types
export interface ChatSession {
  id: string;
  user_id: string;
  session_date: string;
  status: 'active' | 'completed';
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface Message {
  id: string;
  session_id: string;
  sender: 'user' | 'ai';
  content: string;
  created_at: string;
  metadata?: Record<string, unknown>;
  sequence_number: number;
}

export interface SendMessageRequest {
  content: string;
}

export interface SendMessageResponse {
  user_message: Message;
  ai_response: Message;
}

// Analysis types
export interface EmotionalState {
  primary_emotion: string;
  emotions: Record<string, number>;
  confidence: number;
  explanation?: string;
}

export interface BehavioralInsights {
  conversation_patterns: {
    total_length: number;
    estimated_depth: string;
    communication_style: string;
  };
  emotional_progression: {
    primary_emotion: string;
    confidence_level: number;
    emotional_balance: string;
  };
  recommendations: string[];
}

export interface Analysis {
  id: string;
  session_id: string;
  summary: string;
  emotional_state: EmotionalState;
  behavioral_insights: BehavioralInsights;
  tension_score: number;
  relative_score?: number;
  keywords: string[];
  raw_analysis_data?: Record<string, unknown>;
  created_at: string;
}

export interface TensionScoreData {
  date: string;
  tension_score: number;
  relative_score: number;
  session_id: string;
}

export interface TensionStatistics {
  average: number;
  min: number;
  max: number;
  trend: 'improving' | 'declining' | 'stable';
}

export interface TensionScoresResponse {
  scores: TensionScoreData[];
  statistics: TensionStatistics;
}

export interface AnalysisInsight {
  type: string;
  level: 'positive' | 'neutral' | 'attention' | 'high' | 'medium' | 'low';
  message: string;
  value: string | number;
}

export interface AnalysisInsightsResponse {
  insights: AnalysisInsight[];
  timeframe: number;
  statistics: TensionStatistics;
}

// Calendar types
export interface CalendarDay {
  date: string;
  has_session: boolean;
  tension_score?: number;
  status?: string;
  message_count?: number;
}

export interface CalendarMonthData {
  year: number;
  month: number;
  days: CalendarDay[];
}

export interface CalendarResponse {
  month_data: CalendarMonthData;
}

// Session Summary types
export interface SessionSummary {
  id: string;
  date: string;
  status: string;
  message_count: number;
  has_analysis: boolean;
  created_at: string;
  updated_at: string;
}

export interface SessionsResponse {
  sessions: SessionSummary[];
  pagination: {
    total: number;
    limit: number;
    offset: number;
  };
}

// Error types
export interface ErrorDetail {
  code: string;
  message: string;
  details?: unknown;
}

export interface ErrorResponse {
  error: ErrorDetail;
}

// API Response wrapper
export interface ApiResponse<T> {
  data?: T;
  error?: ErrorResponse;
}

// UI State types
export interface LoadingState {
  isLoading: boolean;
  message?: string;
}

export interface NotificationState {
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  id: string;
  duration?: number;
}

// Chart data types for visualization
export interface ChartData {
  labels: string[];
  datasets: {
    label: string;
    data: number[];
    borderColor?: string;
    backgroundColor?: string;
    fill?: boolean;
  }[];
}

export interface TensionChartData extends ChartData {
  trend: 'improving' | 'declining' | 'stable';
  averageScore: number;
} 