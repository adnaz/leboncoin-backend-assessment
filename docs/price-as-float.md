# Item 4: rename `price_cents` to `price` as a float?

No to the float, no to the rename as asked. First I ask the PO and the web/mobile devs what they need, then I propose an additive change.

## Why not a float

Floats can't store most decimal values exactly (`0.1 + 0.2` = `0.30000000000000004`, `int(1.15 * 100)` = `114`). Sums, comparisons and the price filters end up off by a cent, and the client gets the same imprecision because JSON numbers are floats in JavaScript. Money is handled as integers in the smallest unit everywhere (Stripe, Google `Money`), and `price_cents` already is a `BIGINT`. If sub-cent precision is ever needed: fixed-point decimal (`NUMERIC`), sent as a string.

## Why not a rename

`price_cents` is read by the web app and by every installed version of the mobile app, which we can't force to update. A rename is a removal and breaks them. If it must happen: deprecation period agreed with the mobile team and the PO, with an end date.

## What I'd propose

- If the need is display: nothing to change, the clients format the integer in the user's locale.
- If they really want a field named `price`: add it next to `price_cents`, still an integer in cents, mark `price_cents` deprecated in OpenAPI with a date, remove it when no supported app version reads it. No `/v2` for one field; versioning is for changes that can't be additive.

## In this repo

No API change. This file is the answer.
