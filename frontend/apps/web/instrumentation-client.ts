import {
  getWebInstrumentations,
  initializeFaro,
  LogLevel,
} from "@grafana/faro-web-sdk";
import { TracingInstrumentation } from "@grafana/faro-web-tracing";
import { registerInstrumentations } from "@opentelemetry/instrumentation";
import { DocumentLoadInstrumentation } from "@opentelemetry/instrumentation-document-load";

import { parseFaroEnv } from "@/shared/faro";

const faro = parseFaroEnv({
  NEXT_PUBLIC_FARO_URL: process.env.NEXT_PUBLIC_FARO_URL,
  NEXT_PUBLIC_FARO_SAMPLE_RATE: process.env.NEXT_PUBLIC_FARO_SAMPLE_RATE,
});

if (faro.kind === "enabled") {
  initializeFaro({
    url: faro.url,
    app: { name: "web", namespace: "boilerplate" },
    sessionTracking: { samplingRate: faro.sampleRate },
    consoleInstrumentation: {
      disabledLevels: [
        LogLevel.TRACE,
        LogLevel.DEBUG,
        LogLevel.LOG,
        LogLevel.INFO,
      ],
      consoleErrorAsLog: true,
    },
    instrumentations: [
      ...getWebInstrumentations(),
      new TracingInstrumentation(),
    ],
  });

  // Reads the traceparent Next renders from clientTraceMetadata, so the page load joins the SSR trace.
  registerInstrumentations({
    instrumentations: [new DocumentLoadInstrumentation()],
  });
}
