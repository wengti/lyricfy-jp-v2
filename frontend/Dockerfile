# Production image for the Next.js frontend (Cloud Run), using the standalone output.
#
# Build (from frontend/):  docker build -t lyricfy-web .
# Run locally:             docker run --rm -p 3000:3000 \
#                            -e API_URL=http://host.docker.internal:8080 \
#                            -e FRONTEND_API_KEY=... lyricfy-web
#
# API_URL and FRONTEND_API_KEY are read on the server at request time, so they are
# runtime environment variables, not build args: one image works in every environment
# and the key never ends up in an image layer.

# Debian slim rather than Alpine: package-lock.json only carries the glibc build of sharp
# (used by Next's image optimizer).
ARG NODE_IMAGE=node:24-slim

# ---- Build ----
FROM ${NODE_IMAGE} AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1

# Install dependencies in their own layer so code changes don't reinstall them
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
# public/ is empty and untracked, so it may not exist in a fresh checkout
RUN mkdir -p public && npm run build

# ---- Runtime ----
FROM ${NODE_IMAGE}
WORKDIR /app
# server.js binds to HOSTNAME, which Docker sets to the container ID, so listen on all
# interfaces. PORT=3000 is the default for local runs; Cloud Run sets PORT itself.
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    HOSTNAME=0.0.0.0 \
    PORT=3000

# server.js and the traced node_modules, plus the assets standalone doesn't copy itself
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
COPY --from=build --chown=node:node /app/public ./public

# The official node image ships an unprivileged "node" user
USER node
EXPOSE 3000
CMD ["node", "server.js"]
