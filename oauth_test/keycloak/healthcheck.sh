exec 3<>/dev/tcp/localhost/9000; \
echo -en 'GET /health/ready' >&3; \
# Give the server a moment to respond, then search for 'UP'
if timeout 3 cat <&3 | grep -m 1 'UP'; then \
  exec 3<&-; exec 3>&-; exit 0; \
else \
  exec 3<&-; exec 3>&-; exit 1; \
fi
