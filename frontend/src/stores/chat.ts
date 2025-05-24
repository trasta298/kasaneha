import { atom } from 'nanostores';
import type { ChatSession, Message, SendMessageResponse } from '../types';
import { apiClient } from '../api/client';

// Chat state atoms
export const $currentSession = atom<ChatSession | null>(null);
export const $messages = atom<Message[]>([]);
export const $isSending = atom(false);
export const $isLoadingSession = atom(false);
export const $chatError = atom<string | null>(null);

// Chat actions
export const chatActions = {
  async loadTodaySession() {
    $isLoadingSession.set(true);
    $chatError.set(null);

    try {
      // Step 1: Get today's session (GET /sessions/today)
      console.log('Loading today session...');
      const sessionResponse = await apiClient.getTodaySession();
      
      console.log('Session loaded:', sessionResponse.session.id);
      $currentSession.set(sessionResponse.session);
      
      // If there's an initial message from session creation, use it
      if (sessionResponse.initial_message) {
        console.log('Initial message found, using it');
        $messages.set([sessionResponse.initial_message]);
      } else {
        // Step 2: Load existing messages for the session (GET /sessions/:id/messages)
        // This must happen AFTER the session is loaded, never in parallel
        console.log('Loading messages for session:', sessionResponse.session.id);
        const messagesResponse = await apiClient.getSessionMessages(sessionResponse.session.id);
        
        console.log(`Loaded ${messagesResponse.messages.length} messages`);
        $messages.set(messagesResponse.messages);
      }
      
      return sessionResponse;
    } catch (error) {
      console.error('Failed to load today session:', error);
      const message = error instanceof Error ? error.message : 'セッションの取得に失敗しました';
      $chatError.set(message);
      throw error;
    } finally {
      $isLoadingSession.set(false);
    }
  },

  async loadSessionMessages(sessionId: string) {
    $isLoadingSession.set(true);
    $chatError.set(null);

    try {
      // Load session messages (GET /sessions/:id/messages)
      // This endpoint returns both session info and messages
      console.log('Loading session messages for ID:', sessionId);
      const response = await apiClient.getSessionMessages(sessionId);
      
      console.log('Session info loaded:', response.session.id);
      $currentSession.set(response.session);
      
      console.log(`Loaded ${response.messages.length} messages`);
      $messages.set(response.messages);
      
      return response;
    } catch (error) {
      console.error('Failed to load session messages:', error);
      const message = error instanceof Error ? error.message : 'メッセージの取得に失敗しました';
      $chatError.set(message);
      throw error;
    } finally {
      $isLoadingSession.set(false);
    }
  },

  async sendMessage(content: string) {
    const session = $currentSession.get();
    if (!session) {
      throw new Error('アクティブなセッションがありません');
    }

    $isSending.set(true);
    $chatError.set(null);

    try {
      const response = await apiClient.sendMessage(session.id, { content });
      
      // Remove any temporary user messages and replace with server messages
      const currentMessages = $messages.get();
      const messagesWithoutTemp = currentMessages.filter(msg => !msg.id.startsWith('temp-'));
      
      // Add both user message and AI response from server
      $messages.set([
        ...messagesWithoutTemp,
        response.user_message,
        response.ai_response,
      ]);

      return response;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'メッセージの送信に失敗しました';
      $chatError.set(message);
      throw error;
    } finally {
      $isSending.set(false);
    }
  },

  async completeSession() {
    const session = $currentSession.get();
    if (!session) {
      throw new Error('アクティブなセッションがありません');
    }

    $isLoadingSession.set(true);
    $chatError.set(null);

    try {
      const response = await apiClient.completeSession(session.id);
      
      // Update session status to completed
      $currentSession.set({
        ...session,
        status: 'completed',
        completed_at: response.completed_at,
      });

      return response;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'セッションの完了に失敗しました';
      $chatError.set(message);
      throw error;
    } finally {
      $isLoadingSession.set(false);
    }
  },

  clearSession() {
    $currentSession.set(null);
    $messages.set([]);
    $chatError.set(null);
  },

  clearError() {
    $chatError.set(null);
  },

  // Add message to current session (for optimistic updates)
  addMessage(message: Message) {
    const currentMessages = $messages.get();
    $messages.set([...currentMessages, message]);
  },

  // Replace last message (for error corrections)
  replaceLastMessage(message: Message) {
    const currentMessages = $messages.get();
    if (currentMessages.length > 0) {
      const newMessages = [...currentMessages];
      newMessages[newMessages.length - 1] = message;
      $messages.set(newMessages);
    }
  },

  // Remove temporary messages (for error handling)
  removeTempMessages() {
    const currentMessages = $messages.get();
    const filteredMessages = currentMessages.filter(msg => !msg.id.startsWith('temp-'));
    $messages.set(filteredMessages);
  },
}; 