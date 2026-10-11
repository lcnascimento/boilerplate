import { useSyncExternalStore } from "react";

const subscribe = () => () => undefined;

/**
 * False during SSR and hydration, true afterwards. Widgets whose output
 * depends on client-only state (the next-themes theme) render after it.
 */
export function useMounted(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,
    () => false
  );
}
