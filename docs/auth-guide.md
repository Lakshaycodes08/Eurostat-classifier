# Auth & Credentials Guide

How to pass API credentials to `swytchcode exec` and how the kernel handles auth injection. For config file structure, see [config-spec.md](config-spec.md). For debugging auth failures, see [debugging.md](debugging.md).

## How auth headers work

`swytchcode info <canonical_id>` shows the expected auth header under the `Auth` or `HTTP Headers` section:

```
Auth
  Type: api_key
  Header: Authorization
  Format: Bearer <token>
```

When you call `swytchcode exec`, pass the auth header as a top-level arg alongside your payload args. The kernel builds the HTTP request from the wrekenfile contract and merges your args in:

```bash
swytchcode exec stripe.checkout_session_create \
  --body '{"Authorization": "Bearer sk_test_...", "line_items": [...]}'
```

**Never hardcode credentials** in source files. Always read from environment variables.

---

## Credential patterns by auth type

### API key — Bearer token

Most common. The header name is `Authorization` and the format is `Bearer <token>`.

```bash
# In shell
swytchcode exec stripe.checkout_session_create \
  --body "{\"Authorization\": \"Bearer $STRIPE_SECRET_KEY\", ...}"
```

In code (Node.js with swytchcode-runtime):
```js
require('dotenv').config();
const { exec } = require('swytchcode-runtime');

const result = await exec('stripe.checkout_session_create', {
  Authorization: `Bearer ${process.env.STRIPE_SECRET_KEY}`,
  // ... method args
});
```

### API key — custom header

Some services use a non-standard header (e.g. `X-Api-Key`, `Api-Key`). Check `swytchcode info` for the exact header name.

```js
const result = await exec('resend.email_send', {
  'Authorization': `Bearer ${process.env.RESEND_API_KEY}`,
  from: 'noreply@example.com',
  to: 'user@example.com',
  subject: 'Hello',
  html: '<p>Hello</p>',
});
```

### OAuth2

OAuth2 access tokens expire. Refresh the token before calling exec, then pass the current `access_token` as the `Authorization` header. Token refresh is the caller's responsibility — the kernel does not handle it.

```js
const accessToken = await refreshAccessToken(process.env.OAUTH_REFRESH_TOKEN);
const result = await exec('google.calendar_event_create', {
  Authorization: `Bearer ${accessToken}`,
  // ... event args
});
```

---

## Using `.env` files

Store credentials in a `.env` file in your project root. **Never commit `.env` to git** — add it to `.gitignore`.

```bash
# .env
STRIPE_SECRET_KEY=sk_test_...
RESEND_API_KEY=re_...
GOOGLE_ACCESS_TOKEN=ya29....
SWYTCHCODE_TOKEN=swy_key_...
```

Load in Node.js:
```js
require('dotenv').config();
```

Load in Python:
```python
from dotenv import load_dotenv
load_dotenv()
```

For shell scripts, use `direnv` (`.envrc`):
```bash
export STRIPE_SECRET_KEY=sk_test_...
```

---

## Service tokens (`SWYTCHCODE_TOKEN`)

Firebase ID tokens expire after ~1 hour. For CI pipelines, cron jobs, and unattended agents, use a long-lived service token:

1. Obtain a `swy_key_*` token from the SwytchCode dashboard (or your administrator).
2. Set the env var:
   ```bash
   export SWYTCHCODE_TOKEN=swy_key_xxxxxxxxxxxxxxxx
   ```
3. The CLI reads `SWYTCHCODE_TOKEN` before falling back to the Firebase session, so no `swytchcode login` is needed.

In CI (GitHub Actions example):
```yaml
env:
  SWYTCHCODE_TOKEN: ${{ secrets.SWYTCHCODE_TOKEN }}
```

Service tokens are scoped to shell operations only (`swytchcode exec`, `swytchcode get`, etc.). They cannot approve integration upgrades — that requires an interactive user login.

---

## Quick reference: common integrations

| Integration | Header | Env var | Format |
|---|---|---|---|
| Stripe | `Authorization` | `STRIPE_SECRET_KEY` | `Bearer sk_live_...` or `Bearer sk_test_...` |
| Resend | `Authorization` | `RESEND_API_KEY` | `Bearer re_...` |
| SendGrid | `Authorization` | `SENDGRID_API_KEY` | `Bearer SG....` |
| Google Analytics | `Authorization` | `GA_ACCESS_TOKEN` | `Bearer ya29....` |
| OpenAI | `Authorization` | `OPENAI_API_KEY` | `Bearer sk-...` |

Always verify the exact header name and format from `swytchcode info <canonical_id>` — the table above is a guide, not a guarantee.

---

## Auth failure checklist

If `swytchcode exec` returns `category: auth`:

1. **Check the env var is set**: `echo $STRIPE_SECRET_KEY` — if empty, `.env` wasn't loaded.
2. **Check the header name**: run `swytchcode info <canonical_id>` and confirm you're using the exact header name from `Auth` section.
3. **Check the format**: Bearer token? Raw key? The format matters.
4. **Check the key is active**: log into the service dashboard and confirm the key is not revoked or restricted.
5. **Sandbox vs production**: ensure your key matches the `mode` in `tooling.json` (`sandbox` keys hit sandbox endpoints, `production` keys hit live endpoints).
