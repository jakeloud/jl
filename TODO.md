# Todo list for next minor update


1. add multiple domains/per app support.
Main usecase - www. alongside main domain for SEO.

2. split up launch commands into an array of strings.
Why? - 1. clarity 2. clear logging in systemctl 3. less wrapper processes 4. less memory overhead
As a result we could remove liveness checks - just verify that last command has been reached and is running

3. Add version and metadata to API. This is very useful for jakeloud/skill. it has to autoupdate

4. DEBATABLE. add drag and drop upload into /app/data. This can be very useful to share some files, environment variables and such. but this sounds very hackable and unreliable.

5. remove button for docker cache clearance and the functionality altogether. on small machines we can just not use docker. on normal machines we don't need to clear cache agressively - it's actually better to persist it.

6. rework staggered startup of apps (from simple time.Sleep(duration) to real signal from app via go chan)
