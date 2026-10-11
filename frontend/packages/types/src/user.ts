/**
 * The person using the app. Without authentication the app shows a local
 * user, so only `id` and `name` are always present.
 */
export interface User {
  id: string;
  name: string;
  email?: string;
  avatarUrl?: string;
}
