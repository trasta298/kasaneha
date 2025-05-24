import type {
  AuthResponse,
  LoginRequest,
  RegisterRequest,
  User,
  ChatSession,
  Message,
  SendMessageRequest,
  SendMessageResponse,
  SessionsResponse,
  Analysis,
  TensionScoresResponse,
  AnalysisInsightsResponse,
  CalendarResponse,
  ErrorResponse,
} from '../types';

class ApiClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL?: string) {
    // 環境変数から取得（Astroの環境変数）
    const envApiUrl = import.meta.env?.PUBLIC_API_BASE_URL;
    
    // デフォルトはローカル開発用（SPAなのでlocalhost:8080）
    const defaultURL = 'http://localhost:8080';
    
    this.baseURL = baseURL || envApiUrl || defaultURL;
    this.token = this.getStoredToken();
    
    // デバッグログ
    console.log('API Client initialized:', {
      baseURL: this.baseURL,
      envApiUrl,
      defaultURL,
      hasToken: !!this.token
    });
  }

  private getStoredToken(): string | null {
    if (typeof window !== 'undefined') {
      return localStorage.getItem('kasaneha_token');
    }
    return null;
  }

  private setStoredToken(token: string | null): void {
    if (typeof window !== 'undefined') {
      if (token) {
        localStorage.setItem('kasaneha_token', token);
      } else {
        localStorage.removeItem('kasaneha_token');
      }
    }
  }

  setToken(token: string | null): void {
    this.token = token;
    this.setStoredToken(token);
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}/api/v1${endpoint}`;
    
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    console.log('API Request:', { url, method: options.method || 'GET' });

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      // Handle 401 Unauthorized - token is invalid or expired
      if (response.status === 401) {
        console.warn('Unauthorized access, clearing token and redirecting to login');
        this.setToken(null);
        
        // Only redirect if we're in the browser and not already on login page
        if (typeof window !== 'undefined' && !window.location.pathname.includes('/login')) {
          window.location.href = '/login';
        }
      }
      
      const errorData: ErrorResponse = await response.json();
      console.error('API Error:', errorData);
      throw new Error(errorData.error.message || 'API request failed');
    }

    return response.json();
  }

  // Health check
  async health(): Promise<string> {
    const response = await fetch(`${this.baseURL}/api/v1/health`);
    return response.text();
  }

  // Authentication endpoints
  async login(credentials: LoginRequest): Promise<AuthResponse> {
    const response = await this.request<AuthResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
    this.setToken(response.token);
    return response;
  }

  async register(userData: RegisterRequest): Promise<AuthResponse> {
    const response = await this.request<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
    this.setToken(response.token);
    return response;
  }

  async getCurrentUser(): Promise<User> {
    return this.request<User>('/auth/me');
  }

  logout(): void {
    this.setToken(null);
  }

  // Chat session endpoints
  async getTodaySession(): Promise<{
    session: ChatSession;
    initial_message?: Message;
  }> {
    console.log('API: Calling GET /sessions/today');
    const result = await this.request<{
      session: ChatSession;
      initial_message?: Message;
    }>('/sessions/today');
    console.log('API: GET /sessions/today completed, session ID:', result.session.id);
    return result;
  }

  async createSession(date: string): Promise<{ session: ChatSession }> {
    return this.request('/sessions', {
      method: 'POST',
      body: JSON.stringify({ date }),
    });
  }

  async getUserSessions(params?: {
    limit?: number;
    offset?: number;
    year?: number;
    month?: number;
  }): Promise<SessionsResponse> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());
    if (params?.year) searchParams.set('year', params.year.toString());
    if (params?.month) searchParams.set('month', params.month.toString());

    const queryString = searchParams.toString();
    const endpoint = queryString ? `/sessions?${queryString}` : '/sessions';
    
    return this.request(endpoint);
  }

  async getSessionMessages(sessionId: string): Promise<{
    messages: Message[];
    session: ChatSession;
  }> {
    console.log(`API: Calling GET /sessions/${sessionId}/messages`);
    const result = await this.request<{
      messages: Message[];
      session: ChatSession;
    }>(`/sessions/${sessionId}/messages`);
    console.log(`API: GET /sessions/${sessionId}/messages completed, ${result.messages.length} messages loaded`);
    return result;
  }

  async sendMessage(
    sessionId: string,
    message: SendMessageRequest
  ): Promise<SendMessageResponse> {
    return this.request(`/sessions/${sessionId}/messages`, {
      method: 'POST',
      body: JSON.stringify(message),
    });
  }

  async completeSession(sessionId: string): Promise<{
    message: string;
    session_id: string;
    completed_at: string;
  }> {
    return this.request(`/sessions/${sessionId}/complete`, {
      method: 'PUT',
    });
  }

  async getSessionStats(sessionId: string): Promise<{
    message_count: number;
    status: string;
    created_at: string;
    updated_at: string;
    completed_at?: string;
    duration_minutes?: number;
  }> {
    return this.request(`/sessions/${sessionId}/stats`);
  }

  // Analysis endpoints
  async getSessionAnalysis(sessionId: string): Promise<{ analysis: Analysis }> {
    return this.request(`/sessions/${sessionId}/analysis`);
  }

  async triggerSessionAnalysis(sessionId: string): Promise<{ analysis: Analysis }> {
    return this.request(`/sessions/${sessionId}/analysis`, {
      method: 'POST',
    });
  }

  async getTensionScores(days = 30): Promise<TensionScoresResponse> {
    return this.request(`/analysis/scores?days=${days}`);
  }

  async getAnalysisInsights(days = 7): Promise<AnalysisInsightsResponse> {
    return this.request(`/analysis/insights?days=${days}`);
  }

  async getAnalysisHistory(params?: {
    limit?: number;
    offset?: number;
  }): Promise<{
    message: string;
    limit: number;
    offset: number;
  }> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    if (params?.offset) searchParams.set('offset', params.offset.toString());

    const queryString = searchParams.toString();
    const endpoint = queryString ? `/analysis/history?${queryString}` : '/analysis/history';
    
    return this.request(endpoint);
  }

  // Calendar endpoints
  async getCalendarData(year: number, month: number): Promise<CalendarResponse> {
    return this.request(`/calendar/${year}/${month}`);
  }

  // Utility methods
  isAuthenticated(): boolean {
    return this.token !== null;
  }

  getToken(): string | null {
    return this.token;
  }
}

// Export singleton instance
export const apiClient = new ApiClient();
export default apiClient; 