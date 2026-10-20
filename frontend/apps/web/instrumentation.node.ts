import { PinoInstrumentation } from "@opentelemetry/instrumentation-pino";
import { RuntimeNodeInstrumentation } from "@opentelemetry/instrumentation-runtime-node";
import { UndiciInstrumentation } from "@opentelemetry/instrumentation-undici";
import { NodeSDK } from "@opentelemetry/sdk-node";

new NodeSDK({
  instrumentations: [
    new UndiciInstrumentation(),
    new RuntimeNodeInstrumentation(),
    new PinoInstrumentation(),
  ],
}).start();
