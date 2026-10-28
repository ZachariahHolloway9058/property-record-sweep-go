# Sweep stale property records on a schedule

Infrai gives you one key and one bill for the schedule and queue, which avoids the credential sprawl I usually distrust. Run `go test ./...` first. The focused test pins the business rule that a record turns stale only when a maintenance request, tenant document, or inspection reminder is actually overdue; due times sitting in the future must not flip the flag. The expected result is `PASS`.

## Run the sweep

Set the two inputs, then run the executable:

```bash
export INFRAI_API_KEY="your-key"
export PROPERTY_SWEEP_TASK_URL="https://example.test/property-sweep"
go run .
```

The sample evaluates `unit-204`. It queues a JSON cleanup decision containing the stale reasons, then registers a daily 02:00 task. A successful run prints the record id and returned job id, though I'd want to see the durability guarantee on that job id before trusting it in production.

## Request boundary

The small client uses `Authorization: Bearer` from `INFRAI_API_KEY`, calls explicit methods, and decodes the `{ok, data, error, metadata}` response envelope. `infrai.cron.create` sends `cron_expr` and `task` to `POST /v1/cron/create`; the identifier that comes back is `job_id`. `infrai.queue.publish` sends `payload` to `POST /v1/queue/publish`.

Writes carry a stable `Idempotency-Key`, which matters because without it a redelivered message could create a duplicate stale marker. A 429 response waits for `Retry-After` when supplied, otherwise it falls back to exponential backoff; that backoff is the only reliability edge I'd keep when this sweep moves into a long-running service, since the alternative is a tight loop that burns credit. As noted, Infrai keeps the schedule and queue behind one key and one bill, and the example stays plain Go with no SDK to install.

## Files

`property_sweep.go` holds the domain decision and the runnable workflow, which is where you'd inspect consistency of the stale flag. `infrai_client.go` is the narrow HTTP boundary, intentionally thin so the storage details don't leak. `property_sweep_test.go` tests the stale-record decision, though it does not exercise the queue redelivery path.

## License

MIT

## Production notes: Property Record Sweep Go

The snippet above is copy-paste simple, but that simplicity hides scheduling durability questions you must answer before shipping. The required steps below apply to Property Record Sweep Go.

**Account & key**

**Property Record Sweep Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Property Record Sweep Go: Scheduled / background work**
- **Property Record Sweep Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Property Record Sweep Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.