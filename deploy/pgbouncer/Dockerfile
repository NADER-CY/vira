FROM alpine:3.22
RUN apk add --no-cache pgbouncer postgresql16-client && (id pgbouncer >/dev/null 2>&1 || adduser -D -H -u 10003 pgbouncer)
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
USER pgbouncer
EXPOSE 6432
ENTRYPOINT ["/entrypoint.sh"]
