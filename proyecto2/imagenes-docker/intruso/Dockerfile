FROM alpine:latest

CMD ["sh", "-c", "while true; do cat /etc/shadow 2>/dev/null || echo 'Intentando vulnerar lectura...'; sleep 5; done"]