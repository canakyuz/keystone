# Keystone console

The web console for Keystone, the multi-tenant control plane in this repository. Sign in, see your tenant as its own record describes it, manage its members,
and create a tenant while watching the worker provision it.

It is set in the visual language of [canakyuz.co](https://canakyuz.co): sharp surfaces,
hairline rows, and mono type only where the content is machine text.

## What it shows about the API

The console holds no rules of its own. Every page reads through the same API a client
would, and meets the same checks.

- **Membership, not the token.** The role shown and acted on comes from `GET /auth/me`,
  which Keystone reads from the tenant's record on every request. Suspend a member and
  their next click signs them out with a reason.
- **Refusals are shown, not hidden.** A button the API refuses says so in the API's words.
  Controls are offered by role for convenience; the API decides.
- **Provisioning is not a request.** Creating a tenant answers at once with an operation.
  The operation page follows the worker from accepted to active, re-reading every
  1.5 seconds until it ends.
- **One idempotency key per form.** A double click or a retry returns the same operation
  instead of a second tenant.

## How it talks to Keystone

```
browser  ->  Next.js server  ->  Keystone API
             (session cookie)    (Authorization: Bearer)
```

The access token lives in an `httpOnly` cookie and is attached on the server. Nothing in
the browser can read it, and the API needs no cross-origin policy for the console.

## Running it

The API has to be running first; see the [README](../README.md) at the root. Then, from
this directory:

```bash
bun install
cp .env.example .env.local
bun run dev
```

`PORT` picks the port (Next.js defaults to 3000). The development seed
(`make seed-dev`) creates `owner@dev.keystone.local`.

## Stack

Next.js 16 with the App Router and Server Actions, React 19, TypeScript in strict mode,
plain CSS with design tokens, and `bun` as the runner.
