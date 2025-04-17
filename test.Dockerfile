FROM jakeloud-env

RUN mkdir -p /etc/jakeloud /etc/jakeloud/jakeloud-static /var/log/jakeloud

RUN echo '{ \
  "apps": [{ \
    "name": "jakeloud", \
    "port": 666, \
  }], \
  "users": [] \
}' > /etc/jakeloud/conf.json

COPY jl /usr/local/bin/jl

RUN chmod +x /usr/local/bin/jl && \
    chmod -R 777 /etc/jakeloud /etc/nginx /var/log/jakeloud

RUN echo '#!/bin/bash\n\
nginx\n\
dockerd >/var/log/dockerd.log 2>&1 &\n\
/usr/local/bin/jl >>/var/log/jakeloud.log 2>&1 &\n\
tail -f /var/log/jakeloud.log /var/log/dockerd.log' > /entrypoint.sh && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
