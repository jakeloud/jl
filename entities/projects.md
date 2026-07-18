# Projects - new entity

implemented:

1) `projects-root` - `/etc/jakeloud/` for olds. configurable through env var `/app` now
2) "releases" concept: `/app/<project.Name>/r<Release number>` becomes directory for git clone, `/app/<project.Name>` is available to store persistent data. For example, `/app/<project.Name>/data`. Jakeloud maintains two latest releases (current + previous)
3) we don't need to cleanup images and old repos now because of releases. We just remove project dir.
4) Blue-Green deploy. newer release allocates another port. How to do switch? we need to run proxy step after start, the same old way
5) domain becomes optional. If domain not set - no proxying step is required.
6) `docker run --rm` instead of detached docker container. Now jakeloud should redeploy all projects on start. And jakeloud should store logs somewhere. convention `r<Release number>.log` inside project directory feels correct. Log rotation will be figured out later
7) Blue-Green deploy continues. We want to merge build+run steps. Now we run proxy not when build finishes (because we don't know when it finishes), but after a certain timeout if the release is still alive. 5 minutes looks good. Maybe make it configurable per project through UI or via env VAR. Anyway, this is only needed for projects with domain. `example.com:<timeout in minutes>` (for example `example.com:5`) looks like good convention for UI input. Also allow premature confirmation of liveness? Usecase: i look at logs to see that build has finished and click to swith releases in UI.
8) configurable build+start command + $PORT variable (we still allocate port even if we don't proxy). Prefilled with `docker build -t <project.Name> . && docker run -p $PORT:80 --rm <project.Name>`

implementing after a short while:

Refactor:

9) do the frontend
