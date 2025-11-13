## Graceful Shutdown - only finishing the ones in process, drop all other in flight requests

## Load balacing algorithm basic one

## dynamic scaling up & down

## auto scaling up before emergency

## queuing logic if requests exceed what we can handle

## server starting

## healthcheck

## admin endpoint to let user to add, remove, enable, or disable a server

## Notes:

We don't keep dormant servers, we instead remove them directly since they are easy to spin up in local environment.
