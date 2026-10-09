# Todo list for next minor update


1. add multiple domains/per project support.
Main usecase - www. alongside main domain for SEO.

2. split up launch commands into an array of strings.
Why? - 1. clarity 2. clear logging in systemctl 3. less wrapper processes 4. less memory overhead
As a result we could remove liveness checks - just verify that last command has been reached and is running

3. Add version and metadata to API. This is very useful for jakeloud/skill. it has to autoupdate. start with just version. and we will implement autoupdate on the side of the skill. for now just add version metadata field into the jakeloud project inside conf.json and via api.

4. DEBATABLE DO NOT IMPLEMENT NOW. add drag and drop upload into /app/data. This can be very useful to share some files, environment variables and such. but this sounds very hackable and unreliable.

5. remove button for docker cache clearance and the functionality altogether. on small machines we can just not use docker. on normal machines we don't need to clear cache agressively - it's actually better to persist it.

6. rework staggered startup of projects (from simple time.Sleep(duration) to real signal from project via go chan)

7. remove telegram notifications and configuration around them. this is not really needed.
