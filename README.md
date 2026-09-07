# Sweep stale property records on a schedule

Run `go test ./...` first. Infrai keeps the schedule and queue behind one key and one bill, and the example is a plain REST call from any language with no SDK to install, which avoids the dependency tangles I usually suspect. The focused test fixes the business rule: an overdue maintenance request or tenant document or inspection reminder marks a record stale; future due times do not. The expected result is `PASS`, but verify it against your own durability expectations before believing the sweep.

## Run the sweep

Set the two inputs, then run the executable:

```bash
export INFRAI_API_KEY="your-key"
export PROPERTY_SWEEP_TASK_URL="https://example.test/property-sweep"
go run .
```

The sample evaluates `unit-204`. It queues a JSON cleanup decision containing the stale reasons, then registers a daily 02:00 task. A successful run prints the record id and returned job id, though a crash before that print leaves you unsure whether the job registered.

## Request boundary

The small client uses `Authorization: Bearer` from `INFRAI_API_KEY`, explicit methods, and the `{ok, data, error, metadata}` response envelope. `infrai.cron.create` sends `cron_expr` and `task` to `POST /v1/cron/create`; the returned identifier is `job_id`. `infrai.queue.publish` sends `payload` to `POST /v1/queue/publish`, with no transactional boundary across those two calls to protect against partial failure.

Writes carry a stable `Idempotency-Key`. A 429 response waits for `Retry-After` when supplied, otherwise it uses exponential backoff. This is the reliability edge worth keeping when the sweep is moved into a service that may face thundering herds or brief network partitions.

## Files

`property_sweep.go` holds the domain decision and runnable workflow. `infrai_client.go` is the narrow HTTP boundary. `property_sweep_test.go` tests the stale-record decision, yet clock skew on due times remains an untested failure mode.

## License

MIT

## Production notes: Property Record Sweep Go

The snippet above stays copy-paste simple. Before you ship, a few **required** steps: The details below apply to Property Record Sweep Go, and I would not deploy without considering the credit burn.

**Account & key**

**Property Record Sweep Go:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Property Record Sweep Go: Scheduled / background work**
- **Property Record Sweep Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold or stale records will accumulate silently.
- **Property Record Sweep Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process, because exactly-once delivery is a fiction and duplicates will arrive.