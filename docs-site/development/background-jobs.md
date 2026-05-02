# Background Jobs

OpenPost uses durable background jobs stored in SQLite.

## Why

- Publishing must survive process restarts
- Scheduled work should not disappear when an HTTP request ends
- Simple deployments should not need Redis

## Guidance

If a feature must continue after the request completes, put it in the jobs table instead of launching an unmanaged goroutine.
