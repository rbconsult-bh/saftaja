FROM oven/bun:1-alpine AS base
WORKDIR /app
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY . .

FROM base AS dev
EXPOSE 5173
CMD ["bun", "run", "dev", "--host"]

FROM base AS builder
RUN bun run build

FROM caddy:2-alpine AS prod
COPY --from=builder /app/dist /srv
COPY Caddyfile /etc/caddy/Caddyfile
EXPOSE 80
