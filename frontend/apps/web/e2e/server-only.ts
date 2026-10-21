import { registerHooks } from "node:module";

// The fixtures mint Actor tokens with server code guarded by `server-only`. Resolving only that package with the
// react-server condition, as Next does on the server, keeps the rest of the e2e tree (React, lucide-react) on the default one.
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier !== "server-only") return nextResolve(specifier, context);
    return nextResolve(specifier, {
      ...context,
      conditions: [...context.conditions, "react-server"],
    });
  },
});
