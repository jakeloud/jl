FROM debian:11-slim
RUN apt-get update && apt-get install -y systemctl init

COPY jl /usr/local/bin/jl
RUN chmod +x /usr/local/bin/jl

CMD ["/sbin/init"]
