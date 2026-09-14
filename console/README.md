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
- **The history is the trail, not a log.** Each line was written in the transaction that
  made the change, and the table underneath accepts inserts and nothing else. Reading it
  needs the owner or admin role, because a suspended member would like to read it too.

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
(`make seed-dev`) creates `owner@dev.local`.

## Plates

The three engravings are public-domain scans from Wikimedia Commons, cut to one bit by
`scripts/plates.py` (Python 3 with Pillow). The script fetches each scan, refuses it if its
SHA-1 no longer matches, and writes `public/plates/*.svg` and `lib/plates.ts`. Edit the list
in the script and run `python3 scripts/plates.py` from this directory; never edit the output.

| Plate | Page | Source |
|---|---|---|
| The vault on its centering | Sign in | Viollet-le-Duc, *Dictionnaire raisonné de l'architecture*, 1856, [Commons](https://commons.wikimedia.org/wiki/File:Construction.voute.romaine.png) |
| One pier, two arches | Tenant | Viollet-le-Duc, *Dictionnaire raisonné de l'architecture*, 1856, [Commons](https://commons.wikimedia.org/wiki/File:Tas.de.charge.2.png) |
| The work is done in the yard | New tenant | Diderot and d'Alembert, *Encyclopédie*, Maçonnerie plate I, 1762, [Commons](https://commons.wikimedia.org/wiki/File:Engraving_from_Diderot%2C_Encyclop%C3%A9die%2C_v._1%2C_pl._194%2C_Architecture_Maconnerie._Masonry_arch._LCCN2006677828.jpg) |

## Stack

Next.js 16 with the App Router and Server Actions, React 19, TypeScript in strict mode,
plain CSS with design tokens, and `bun` as the runner.
