# フロントエンド設計

## 技術スタック

- **Astro 4.x**: 静的サイト生成＋ハイドレーション
- **TypeScript**: 型安全性の確保
- **Tailwind CSS**: ユーティリティファーストCSS
- **Chart.js**: データ可視化
- **PWA**: プログレッシブWebアプリ機能

## プロジェクト構造

```
frontend/
├── src/
│   ├── components/          # 再利用可能コンポーネント
│   │   ├── ui/             # 基本UIコンポーネント
│   │   ├── chat/           # チャット関連コンポーネント
│   │   ├── calendar/       # カレンダー関連コンポーネント
│   │   └── analytics/      # 分析・グラフコンポーネント
│   ├── layouts/            # レイアウトコンポーネント
│   ├── pages/              # ページコンポーネント
│   ├── stores/             # 状態管理
│   ├── utils/              # ユーティリティ関数
│   ├── types/              # TypeScript型定義
│   └── styles/             # グローバルスタイル
├── public/                 # 静的ファイル
│   ├── icons/              # PWAアイコン
│   ├── manifest.json       # PWAマニフェスト
│   └── sw.js              # Service Worker
└── astro.config.mjs        # Astro設定
```

## ページ設計

### 1. チャット画面 (`/`)
メインのチャットインターフェース

```typescript
// src/pages/index.astro
---
import Layout from '../layouts/Layout.astro';
import ChatContainer from '../components/chat/ChatContainer.astro';
---

<Layout title="今日の日記">
  <ChatContainer />
</Layout>
```

### 2. 履歴画面 (`/history`)
過去のチャット履歴一覧

```typescript
// src/pages/history.astro
---
import Layout from '../layouts/Layout.astro';
import HistoryList from '../components/history/HistoryList.astro';
---

<Layout title="履歴">
  <HistoryList />
</Layout>
```

### 3. カレンダー画面 (`/calendar`)
月次カレンダー表示

```typescript
// src/pages/calendar.astro
---
import Layout from '../layouts/Layout.astro';
import CalendarView from '../components/calendar/CalendarView.astro';
---

<Layout title="カレンダー">
  <CalendarView />
</Layout>
```

### 4. 分析画面 (`/analytics`)
テンションスコアとグラフ表示

```typescript
// src/pages/analytics.astro
---
import Layout from '../layouts/Layout.astro';
import AnalyticsDashboard from '../components/analytics/AnalyticsDashboard.astro';
---

<Layout title="分析">
  <AnalyticsDashboard />
</Layout>
```

## コンポーネント設計

### 1. チャット関連コンポーネント

#### ChatContainer
```typescript
// src/components/chat/ChatContainer.astro
---
import { chatStore } from '../../stores/chatStore';
import MessageList from './MessageList.astro';
import MessageInput from './MessageInput.astro';
import SessionHeader from './SessionHeader.astro';
---

<div class="h-screen flex flex-col bg-gradient-to-br from-blue-50 to-purple-50">
  <SessionHeader />
  <MessageList />
  <MessageInput />
</div>

<script>
  import { initializeChat } from '../../utils/chatUtils';
  initializeChat();
</script>
```

#### MessageBubble
```typescript
// src/components/chat/MessageBubble.astro
---
export interface Props {
  message: string;
  sender: 'user' | 'ai';
  timestamp: string;
  isLatest?: boolean;
}

const { message, sender, timestamp, isLatest = false } = Astro.props;
---

<div class={`flex ${sender === 'user' ? 'justify-end' : 'justify-start'} mb-4`}>
  <div class={`
    max-w-xs lg:max-w-md px-4 py-2 rounded-2xl
    ${sender === 'user' 
      ? 'bg-blue-500 text-white rounded-br-md' 
      : 'bg-white border border-gray-200 text-gray-800 rounded-bl-md shadow-sm'
    }
    ${isLatest ? 'animate-slide-up' : ''}
  `}>
    <p class="text-sm whitespace-pre-wrap">{message}</p>
    <span class={`text-xs mt-1 block ${sender === 'user' ? 'text-blue-100' : 'text-gray-500'}`}>
      {new Date(timestamp).toLocaleTimeString('ja-JP', { 
        hour: '2-digit', 
        minute: '2-digit' 
      })}
    </span>
  </div>
</div>
```

#### MessageInput
```typescript
// src/components/chat/MessageInput.astro
---
---

<div class="p-4 bg-white border-t border-gray-200">
  <form id="message-form" class="flex gap-2">
    <input 
      type="text" 
      id="message-input"
      placeholder="メッセージを入力..."
      class="flex-1 px-4 py-2 border border-gray-300 rounded-full focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      autocomplete="off"
    />
    <button 
      type="submit"
      class="px-6 py-2 bg-blue-500 text-white rounded-full hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
      id="send-button"
    >
      送信
    </button>
  </form>
</div>

<script>
  import { sendMessage } from '../../utils/chatUtils';
  
  const form = document.getElementById('message-form') as HTMLFormElement;
  const input = document.getElementById('message-input') as HTMLInputElement;
  const button = document.getElementById('send-button') as HTMLButtonElement;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const message = input.value.trim();
    if (!message) return;

    button.disabled = true;
    input.value = '';
    
    try {
      await sendMessage(message);
    } finally {
      button.disabled = false;
      input.focus();
    }
  });
</script>
```

### 2. カレンダーコンポーネント

#### CalendarView
```typescript
// src/components/calendar/CalendarView.astro
---
import CalendarGrid from './CalendarGrid.astro';
import CalendarHeader from './CalendarHeader.astro';
---

<div class="max-w-md mx-auto bg-white rounded-lg shadow-lg overflow-hidden">
  <CalendarHeader />
  <CalendarGrid />
</div>

<script>
  import { calendarStore } from '../../stores/calendarStore';
  import { loadCalendarData } from '../../utils/calendarUtils';
  
  // 初期データ読み込み
  loadCalendarData();
</script>
```

#### CalendarDay
```typescript
// src/components/calendar/CalendarDay.astro
---
export interface Props {
  date: string;
  hasSession: boolean;
  tensionScore?: number;
  isToday?: boolean;
  isSelected?: boolean;
}

const { date, hasSession, tensionScore, isToday = false, isSelected = false } = Astro.props;
---

<button 
  class={`
    w-full h-12 flex flex-col items-center justify-center text-xs
    transition-colors duration-200
    ${isToday ? 'bg-blue-500 text-white' : ''}
    ${isSelected ? 'ring-2 ring-blue-300' : ''}
    ${hasSession ? 'hover:bg-gray-50' : 'text-gray-400'}
  `}
  data-date={date}
  disabled={!hasSession}
>
  <span class="font-medium">{new Date(date).getDate()}</span>
  {hasSession && tensionScore && (
    <div class={`
      w-2 h-2 rounded-full mt-1
      ${tensionScore >= 80 ? 'bg-green-400' : 
        tensionScore >= 60 ? 'bg-yellow-400' : 
        tensionScore >= 40 ? 'bg-orange-400' : 'bg-red-400'
      }
    `}></div>
  )}
</button>
```

### 3. 分析・グラフコンポーネント

#### TensionChart
```typescript
// src/components/analytics/TensionChart.astro
---
---

<div class="bg-white rounded-lg shadow-lg p-6">
  <h3 class="text-lg font-semibold text-gray-800 mb-4">テンションスコア推移</h3>
  <div class="h-64">
    <canvas id="tension-chart"></canvas>
  </div>
</div>

<script>
  import Chart from 'chart.js/auto';
  import { tensionScoreStore } from '../../stores/analyticsStore';
  
  let chart: Chart | null = null;
  
  function createChart(data: any[]) {
    const ctx = document.getElementById('tension-chart') as HTMLCanvasElement;
    
    if (chart) {
      chart.destroy();
    }
    
    chart = new Chart(ctx, {
      type: 'line',
      data: {
        labels: data.map(d => d.date),
        datasets: [{
          label: 'テンションスコア',
          data: data.map(d => d.tension_score),
          borderColor: 'rgb(59, 130, 246)',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          borderWidth: 2,
          fill: true,
          tension: 0.4
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        scales: {
          y: {
            beginAtZero: true,
            max: 100
          }
        },
        plugins: {
          legend: {
            display: false
          }
        }
      }
    });
  }
  
  // データの監視と更新
  tensionScoreStore.subscribe(createChart);
</script>
```

## 状態管理

### Astroストア設計

```typescript
// src/stores/chatStore.ts
import { atom, map } from 'nanostores';

export interface Message {
  id: string;
  content: string;
  sender: 'user' | 'ai';
  timestamp: string;
}

export interface ChatSession {
  id: string;
  date: string;
  status: 'active' | 'completed';
  messages: Message[];
}

export const currentSession = atom<ChatSession | null>(null);
export const messages = atom<Message[]>([]);
export const isLoading = atom(false);
export const error = atom<string | null>(null);

// アクション
export function addMessage(message: Message) {
  messages.set([...messages.get(), message]);
}

export function setCurrentSession(session: ChatSession) {
  currentSession.set(session);
  messages.set(session.messages);
}

export function setLoading(loading: boolean) {
  isLoading.set(loading);
}

export function setError(errorMessage: string | null) {
  error.set(errorMessage);
}
```

## API クライアント

```typescript
// src/utils/apiClient.ts
class ApiClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    this.token = localStorage.getItem('auth_token');
  }

  private async request<T>(
    endpoint: string, 
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
      ...(this.token && { Authorization: `Bearer ${this.token}` }),
      ...options.headers,
    };

    const response = await fetch(url, {
      ...options,
      headers,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.message || `HTTP ${response.status}`);
    }

    return response.json();
  }

  // チャット関連
  async getTodaySession() {
    return this.request<{ session: ChatSession | null }>('/sessions/today');
  }

  async createSession(date: string) {
    return this.request<{ session: ChatSession; initial_message: Message }>(
      '/sessions', 
      {
        method: 'POST',
        body: JSON.stringify({ date }),
      }
    );
  }

  async sendMessage(sessionId: string, content: string) {
    return this.request<{ user_message: Message; ai_response: Message }>(
      `/sessions/${sessionId}/messages`,
      {
        method: 'POST',
        body: JSON.stringify({ content }),
      }
    );
  }

  // 分析関連
  async getSessionAnalysis(sessionId: string) {
    return this.request<{ analysis: Analysis | null }>(
      `/sessions/${sessionId}/analysis`
    );
  }

  async getTensionScores(params: { start_date?: string; end_date?: string } = {}) {
    const query = new URLSearchParams(params as any).toString();
    return this.request<{ scores: TensionScore[]; statistics: Statistics }>(
      `/analysis/scores?${query}`
    );
  }
}

export const apiClient = new ApiClient(
  import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api/v1'
);
```

## PWA設定

### Service Worker
```javascript
// public/sw.js
const CACHE_NAME = 'kasaneha-v1';
const urlsToCache = [
  '/',
  '/history',
  '/calendar',
  '/analytics',
  '/styles/global.css',
  // 重要なアセット
];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(urlsToCache))
  );
});

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request)
      .then((response) => {
        // キャッシュにあればそれを返す
        if (response) {
          return response;
        }
        return fetch(event.request);
      })
  );
});
```

### PWA Manifest
```json
{
  "name": "Kasaneha - AI日記アプリ",
  "short_name": "Kasaneha",
  "description": "AIと一緒に書く日記アプリ",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#3b82f6",
  "icons": [
    {
      "src": "/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

## レスポンシブデザイン

### Tailwind設定
```javascript
// tailwind.config.mjs
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  theme: {
    extend: {
      animation: {
        'slide-up': 'slideUp 0.3s ease-out',
        'fade-in': 'fadeIn 0.5s ease-out',
      },
      keyframes: {
        slideUp: {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
      },
      colors: {
        kasane: {
          50: '#f0f9ff',
          500: '#3b82f6',
          600: '#2563eb',
        },
      },
    },
  },
  plugins: [],
}
```

## アクセシビリティ

### キーボードナビゲーション
```typescript
// src/utils/accessibility.ts
export function setupKeyboardNavigation() {
  document.addEventListener('keydown', (e) => {
    // Enterキーでメッセージ送信
    if (e.key === 'Enter' && !e.shiftKey) {
      const activeElement = document.activeElement as HTMLElement;
      if (activeElement?.id === 'message-input') {
        e.preventDefault();
        const form = document.getElementById('message-form') as HTMLFormElement;
        form.dispatchEvent(new Event('submit'));
      }
    }
    
    // Escキーでモーダル閉じる
    if (e.key === 'Escape') {
      const modal = document.querySelector('[data-modal]') as HTMLElement;
      if (modal) {
        modal.style.display = 'none';
      }
    }
  });
}
```

### ARIA属性
```html
<!-- src/components/chat/MessageList.astro -->
<div 
  role="log" 
  aria-live="polite" 
  aria-label="チャット履歴"
  class="flex-1 overflow-y-auto p-4"
>
  {messages.map((message) => (
    <div aria-label={`${message.sender === 'user' ? 'あなた' : 'かさね'}のメッセージ`}>
      <!-- メッセージコンテンツ -->
    </div>
  ))}
</div>
``` 