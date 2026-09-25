# Render deployment

The service requires `MONGODB_URI` and `JWT_SECRET` at startup. Render does not
load the local `.env` file, and `.env` is intentionally excluded from Git.

Set these environment variables in the Render service:

- `MONGODB_URI`: the MongoDB Atlas connection string. In Atlas, add Render's
  outbound access (or `0.0.0.0/0` temporarily for testing) to the network
  access allowlist.
- `JWT_SECRET`: a long random value. Do not reuse the local development value.
- `MONGODB_DATABASE`: `pulsepoll` (optional; this is the default).
- `FRONTEND_ORIGIN`: the deployed frontend URL, without a trailing slash.
- `REDIS_URL`: optional. Leave unset if realtime streaming is not required.

The Render service should use:

- Build command: `go build -o app .`
- Start command: `./app`

After saving the variables, trigger a new deploy. A successful startup includes
`MongoDB connected successfully!` in the deploy logs.

## Authentication failures

If the log contains `bad auth` or `authentication failed`, the Render
`MONGODB_URI` has invalid MongoDB Atlas credentials. This is not caused by the
Go application:

1. In MongoDB Atlas, open **Database Access** and reset the password for the
   database user used by this service.
2. In Atlas, choose **Connect → Drivers**, select Go, and copy a new connection
   string. Replace the placeholders with that database user's credentials.
3. URL-encode reserved characters in the username or password. For example,
   `@` becomes `%40`, `:` becomes `%3A`, and `/` becomes `%2F`.
4. Replace the Render `MONGODB_URI` value with the new string and save it.
5. Confirm the Atlas Network Access list allows Render to connect, then deploy
   again.

Do not commit the connection string or paste it into source code. If the URI
contains `&`, keep it as one Render environment-variable value; do not add
shell quotes around it.
