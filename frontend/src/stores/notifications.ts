import { atom } from 'nanostores';
import type { NotificationState } from '../types';

// Notifications state
export const $notifications = atom<NotificationState[]>([]);

// Notification actions
export const notificationActions = {
  add(notification: Omit<NotificationState, 'id'>) {
    const id = Date.now().toString();
    const newNotification: NotificationState = {
      ...notification,
      id,
      duration: notification.duration || 5000,
    };

    const currentNotifications = $notifications.get();
    $notifications.set([...currentNotifications, newNotification]);

    // Auto-remove after duration
    if (newNotification.duration && newNotification.duration > 0) {
      setTimeout(() => {
        this.remove(id);
      }, newNotification.duration);
    }

    return id;
  },

  remove(id: string) {
    const currentNotifications = $notifications.get();
    $notifications.set(currentNotifications.filter(n => n.id !== id));
  },

  clear() {
    $notifications.set([]);
  },

  // Convenience methods
  success(message: string, duration?: number) {
    return this.add({ type: 'success', message, duration });
  },

  error(message: string, duration?: number) {
    return this.add({ type: 'error', message, duration });
  },

  warning(message: string, duration?: number) {
    return this.add({ type: 'warning', message, duration });
  },

  info(message: string, duration?: number) {
    return this.add({ type: 'info', message, duration });
  },
}; 