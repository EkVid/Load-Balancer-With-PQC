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

When hitting the LB with a burst of requests (a total amount of requests that exceed the global limit), the expected number of successful requests may vary. This is normal behavior, and it happens because of several timing-dependent factors:

- Rate-limiter window timing: If the burst arrives slightly earlier or later within the 1-second window, a different number of requests will be allowed before the limiter cuts off.

- Concurrency availability: Depending on how quickly ongoing requests finish, the number of available concurrency slots will differ from run to run, allowing more (or fewer) new requests to slip in.

- Load-test burst irregularity: Load generators don’t send all requests at the exact same microsecond; small timing differences (even a few ms) cause different outcomes under rate limiting.

- Backend processing variations: Minor differences in response time (GC, CPU scheduling, goroutine timing) affect how fast concurrency frees up, which directly changes how many requests get accepted.
