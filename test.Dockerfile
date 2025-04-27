FROM debian:11-slim

RUN echo '#!/bin/bash\n\
/usr/local/bin/jl' > /entrypoint.sh && \
    chmod +x /entrypoint.sh

COPY jl /usr/local/bin/jl
RUN chmod +x /usr/local/bin/jl

ENTRYPOINT ["/entrypoint.sh"]
