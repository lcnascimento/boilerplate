import { getIam } from "@/shared/auth/auth";

function handle(request: Request): Promise<Response> {
  return getIam().auth.handler(request);
}

export { handle as GET, handle as POST };
