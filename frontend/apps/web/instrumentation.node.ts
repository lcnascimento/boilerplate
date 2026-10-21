import { PinoInstrumentation } from "@opentelemetry/instrumentation-pino";
import { RuntimeNodeInstrumentation } from "@opentelemetry/instrumentation-runtime-node";
import { UndiciInstrumentation } from "@opentelemetry/instrumentation-undici";
import { NodeSDK } from "@opentelemetry/sdk-node";

import { seedLoginFailures } from "@/shared/auth/login-failures";

new NodeSDK({
  instrumentations: [
    new UndiciInstrumentation(),
    new RuntimeNodeInstrumentation(),
    new PinoInstrumentation(),
  ],
}).start();

seedLoginFailures();
