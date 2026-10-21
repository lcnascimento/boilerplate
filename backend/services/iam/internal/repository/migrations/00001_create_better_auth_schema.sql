-- +goose Up
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';

CREATE TABLE "user" (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL UNIQUE,
    "emailVerified" boolean NOT NULL,
    image text,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE session (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    "expiresAt" timestamptz NOT NULL,
    token text NOT NULL UNIQUE,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL,
    "ipAddress" text,
    "userAgent" text,
    "userId" uuid NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    "activeOrganizationId" text
);

CREATE TABLE account (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    "accountId" text NOT NULL,
    "providerId" text NOT NULL,
    "userId" uuid NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    "accessToken" text,
    "refreshToken" text,
    "idToken" text,
    "accessTokenExpiresAt" timestamptz,
    "refreshTokenExpiresAt" timestamptz,
    scope text,
    password text,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL
);

CREATE TABLE verification (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    identifier text NOT NULL,
    value text NOT NULL,
    "expiresAt" timestamptz NOT NULL,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE jwks (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    "publicKey" text NOT NULL,
    "privateKey" text NOT NULL,
    "createdAt" timestamptz NOT NULL,
    "expiresAt" timestamptz,
    alg text,
    crv text
);

CREATE TABLE organization (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    logo text,
    "createdAt" timestamptz NOT NULL,
    metadata text
);

CREATE TABLE member (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    "organizationId" uuid NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    "userId" uuid NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
    role text NOT NULL CONSTRAINT member_role_check CHECK (role IN ('owner', 'admin', 'viewer')),
    "createdAt" timestamptz NOT NULL
);

CREATE TABLE invitation (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    "organizationId" uuid NOT NULL REFERENCES organization (id) ON DELETE CASCADE,
    email text NOT NULL,
    role text CONSTRAINT invitation_role_check CHECK (role IN ('owner', 'admin', 'viewer')),
    status text NOT NULL,
    "expiresAt" timestamptz NOT NULL,
    "createdAt" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "inviterId" uuid NOT NULL REFERENCES "user" (id) ON DELETE CASCADE
);

CREATE TABLE "rateLimit" (
    id uuid NOT NULL DEFAULT pg_catalog.gen_random_uuid() PRIMARY KEY,
    key text NOT NULL UNIQUE,
    count bigint NOT NULL,
    "lastRequest" bigint NOT NULL
);

CREATE INDEX "session_userId_idx" ON session ("userId");

CREATE INDEX "account_userId_idx" ON account ("userId");

CREATE INDEX "verification_identifier_idx" ON verification (identifier);

CREATE INDEX "member_organizationId_idx" ON member ("organizationId");

CREATE INDEX "member_userId_idx" ON member ("userId");

CREATE INDEX "invitation_organizationId_idx" ON invitation ("organizationId");

CREATE INDEX "invitation_email_idx" ON invitation (email);

-- +goose Down
SET LOCAL lock_timeout = '1s';
SET LOCAL statement_timeout = '5s';
-- squawk-ignore ban-drop-table
DROP TABLE "rateLimit";
-- squawk-ignore ban-drop-table
DROP TABLE invitation;
-- squawk-ignore ban-drop-table
DROP TABLE member;
-- squawk-ignore ban-drop-table
DROP TABLE organization;
-- squawk-ignore ban-drop-table
DROP TABLE jwks;
-- squawk-ignore ban-drop-table
DROP TABLE verification;
-- squawk-ignore ban-drop-table
DROP TABLE account;
-- squawk-ignore ban-drop-table
DROP TABLE session;
-- squawk-ignore ban-drop-table
DROP TABLE "user";
