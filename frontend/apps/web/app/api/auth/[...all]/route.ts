import { getIam } from "@/shared/auth/auth";
import { withLoginFailures } from "@/shared/auth/login-failures";

const handle = withLoginFailures((request) => getIam().auth.handler(request));

export { handle as GET, handle as POST };
