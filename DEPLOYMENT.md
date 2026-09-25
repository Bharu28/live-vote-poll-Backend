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
