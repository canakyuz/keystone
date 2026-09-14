# Product

## Register

product

## Users

An engineer evaluating the Keystone repository, usually a hiring engineer with about ten
minutes. They have read the README's claims (schema-per-tenant isolation, durable
provisioning, membership checked on every request) and open the console to see those
claims hold from a browser. They are fluent in good tools and notice anything that is
subtly off. A second, smaller audience is an operator running a Keystone control plane
for real, doing the same tasks for real.

## Product Purpose

The console makes the API's guarantees visible. It holds no rules of its own: every page
reads through the same API a client would, and shows refusals in the API's words.
Success is the evaluator concluding, from the interface alone, that the system is
careful: provisioning is watched rather than awaited, the history is a trail rather
than a log, and a suspended member is signed out on their next click.

## Brand Personality

Precise, candid, engineered. The voice of a well-kept instrument's manual: plain
sentences that say what happened and why, no salesmanship. It shares the visual
language of canakyuz.co (sharp surfaces, hairline rules, engraved plates, mono only
for machine text), so the console reads as the same author's work.

## Anti-references

- The generic SaaS admin template: left sidebar, a grid of rounded metric cards,
  gradient accents, an icon beside every label.
- Marketing tone inside a tool: hero animation, sales copy, confetti on success.
- Dashboards that decorate data they do not have (fake charts, sparklines of nothing).
- Coloured side-stripe callouts and pill badges as default scaffolding.

## Design Principles

1. **Show the mechanism.** Every screen should expose what the API actually did:
   the operation, the key, the refusal, the record it read.
2. **Refusals are content.** An error is information about the system's rules, shown
   plainly and never hidden behind a generic toast.
3. **Plain words over labels.** Sentences that explain, in the reader's vocabulary.
4. **One loud thing per screen.** Hierarchy comes from a single strong element and
   quiet, consistent structure around it.
5. **Earned familiarity.** Standard affordances, the same control vocabulary on every
   screen; character comes from type, rules and plates, not invented widgets.

## Accessibility & Inclusion

WCAG 2.1 AA. Full keyboard use with visible focus, a skip link, reduced motion
respected, status never carried by colour alone (tags pair colour with a word), and
layouts that hold at phone width without horizontal page scroll.
