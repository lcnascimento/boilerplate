import { useSyncExternalStore } from "react";

const subscribe = () => () => undefined;

export function useMounted(): boolean {
  return useSyncExternalStore(
    subscribe,
    () => true,
    () => false
  );
}
