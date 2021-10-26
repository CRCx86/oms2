FROM caddy:2-alpine
COPY api/caddy.json /etc/caddy/caddy.json
COPY api/. /var/www/.