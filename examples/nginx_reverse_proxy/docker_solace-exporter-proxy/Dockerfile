FROM dockerio.docker.bin.sbb.ch/library/nginx:1.21-alpine

RUN apk update  &&  apk add inotify-tools bash tini

WORKDIR /opt
COPY watch-nginx.sh .
RUN  chmod +x  watch-nginx.sh && \
     chgrp -R root /var/cache/nginx /var/run /var/log/nginx && \
     chmod -R 770 /var/cache/nginx /var/run /var/log/nginx

EXPOSE 9000

# Run your program under Tini
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/opt/watch-nginx.sh"]