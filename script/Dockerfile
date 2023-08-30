# TODO: Check later
FROM alpine:latest
RUN apk add --no-cache bash
RUN apk add --no-cache netcat-openbsd
COPY ./test.sh /test.sh
RUN chmod +x /test.sh
CMD ["/test.sh"]
