# Setup procedure

Why we need it?
- install required packages (certbot, nginx)
- add ability to install docker optionally
- add service script to autolauch jl after restart
- create /etc/jakeloud

- print sslip domain to console to Ctrl+Click into
set up application

## Logic
1. check all individual packages for presense
2. install required packages and setup required files
3. ask for additional deps one by one

## How to activate
Activate setup by default. Use -d flag to launch in
daemon mode (which is previous default).
