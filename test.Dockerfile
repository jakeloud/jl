FROM debian:11-slim

COPY jl /usr/local/bin/jl
RUN chmod +x /usr/local/bin/jl
