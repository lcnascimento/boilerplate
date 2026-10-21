import createNextIntlPlugin from "next-intl/plugin";

import type { NextConfig } from "next";

const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const nextConfig: NextConfig = {
  // `next dev` writes an AGENTS.md when it detects an AI agent.
  agentRules: false,
  reactCompiler: true,
  typedRoutes: true,
  experimental: {
    clientTraceMetadata: ["traceparent"],
  },
  transpilePackages: [
    "@boilerplate/api",
    "@boilerplate/i18n",
    "@boilerplate/types",
    "@boilerplate/ui",
    "@boilerplate/utils",
  ],
};

export default withNextIntl(nextConfig);
