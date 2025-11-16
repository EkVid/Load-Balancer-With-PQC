## Graceful Shutdown - only finishing the ones in process, drop all other in flight requests

## Load balacing algorithm basic one

## dynamic scaling up & down

## auto scaling up before emergency

## queuing logic if requests exceed what we can handle

## server starting

## healthcheck

## admin endpoint to let user to add, remove, enable, or disable a server

## requests with priority handling

## Notes:

We don't keep dormant servers, we instead remove them directly since they are easy to spin up in local environment.

If a user wants a req to be prioritized, it should do sth like: -H "Priority: High"

To use priority, we can run the testing files

High prioroty ones are handled immediately and low priority ones are enqueued first. So the lowqueueLength is roughly how many low priority ones it can handle, wheras highqueueLength is the queue to store the overflowing high priority ones, so the lb can handle roughly (highqueueLength + maxConcurrent) number of high priority requests
