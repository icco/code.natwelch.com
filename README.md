# code.natwelch.com

This should show my contributions to software overtime.

## Desired Graphs

 * Watchers over time
 * Forks over time
 * Repositories over time
 * commits per day

## Notes

 * http://stackoverflow.com/questions/8301531/dealing-with-dates-on-d3-js-axis
 * For installing on OSX:

```
$ brew install v8@3.15
$ bundle config build.libv8 --with-system-v8
$ bundle config build.therubyracer --with-v8-dir=$(brew --prefix v8@3.15)
$ bundle install
```

## Deployment

Because I'm an idiot and lazy, this is actually running on two seperate pieces of infrastructure.

 - Heroku: Postgres, Memcachier, Cron Jobs
 - GKE: Web server
