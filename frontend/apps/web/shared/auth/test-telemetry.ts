import { metrics, NodeSDK, tracing } from "@opentelemetry/sdk-node";

class CollectingReader extends metrics.MetricReader {
  protected onForceFlush(): Promise<void> {
    return Promise.resolve();
  }

  protected onShutdown(): Promise<void> {
    return Promise.resolve();
  }
}

export interface TestTelemetry {
  counter(name: string): Promise<Record<string, number>>;
  spans(): Promise<tracing.ReadableSpan[]>;
}

// Starts the same SDK as instrumentation.node.ts, reading in memory instead of exporting over OTLP.
export function startTestTelemetry(): TestTelemetry {
  const reader = new CollectingReader();
  const exporter = new tracing.InMemorySpanExporter();
  const processor = new tracing.SimpleSpanProcessor(exporter);
  new NodeSDK({
    metricReaders: [reader],
    spanProcessors: [processor],
    logRecordProcessors: [],
  }).start();

  return {
    async counter(name) {
      const { resourceMetrics } = await reader.collect();
      const points: Record<string, number> = {};
      for (const scope of resourceMetrics.scopeMetrics) {
        for (const metric of scope.metrics) {
          if (metric.descriptor.name !== name) continue;
          for (const point of metric.dataPoints) {
            const labels = Object.entries(point.attributes)
              .map(([key, value]) => `${key}=${String(value)}`)
              .join(",");
            points[labels] = Number(point.value);
          }
        }
      }
      return points;
    },
    async spans() {
      await processor.forceFlush();
      return exporter.getFinishedSpans();
    },
  };
}
