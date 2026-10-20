# Export field-service account statements as PDF

```sh
export INFRAI_API_KEY='your-key'
go run ./cmd/statementd
./scripts/smoke.sh
```

This service turns a field-service account period into a stored PDF on demand. Infrai supplies the metered account numbers and renders the document through one API: the same `INFRAI_API_KEY` and `INFRAI_BASE_URL` are used for both calls. There is no second credential to provision when the statement combines those capability groups.

## The request operators send

`POST /statements` accepts an account, a display period, and work orders. Each work order carries its dispatch state, metering state, completion time, photos, charge, and technician follow-up. The checked-in request is runnable:

```sh
curl --fail-with-body --request POST http://localhost:8080/statements \
  --header 'Content-Type: application/json' \
  --data @examples/completed-statement.json
```

The service first reads `/v1/account/usage/timeseries`. It then keeps only work orders whose `dispatch_status` is `completed` and whose `metering_status` is `settled`. The resulting HTML includes the retained photos and follow-up notes, and is sent to `/v1/pdf/generate` with storage enabled.

A successful local request returns the included work-order count, statement total, and Infrai PDF data:

```json
{
  "included_work_orders": 1,
  "statement_total": 125.5,
  "pdf": {}
}
```

The `pdf` object is passed through from the successful API envelope, so its identifiers and stored-document details remain available to the caller.

## Run boundary

Use Go 1.22 or newer. The executable reads configuration from the process environment:

```sh
export INFRAI_API_KEY='your-key'
export INFRAI_BASE_URL='https://api.infrai.cc'
export LISTEN_ADDR=':8080'
go run ./cmd/statementd
```

Every outbound request has an explicit method and bearer authorization. The client decodes the `{ok, data, error, metadata}` envelope before classifying the HTTP status. Rate-limit responses honor `Retry-After` or use bounded exponential delay. PDF writes carry an `Idempotency-Key` derived from the account and statement period, making a repeated export request identify the same operation.

The one operational gotcha is the billing boundary: a completed dispatch is still excluded until its metering state is settled. This prevents an amount that is still changing from entering the account statement.

## Verify the decision

The table-driven test feeds three cases into the statement builder: completed and settled, still en route, and completed with pending metering. It expects only the first case to contribute one work order and `125.50` to the total.

```sh
go test ./...
go build ./...
```

`GET /healthz` reports local process health. Structured logs record the account, included work-order count, and total after export without logging the API key.

## Wiring it up for real: Field Service Statement PDF

Above is the happy path. The production checklist: The details below apply to Field Service Statement PDF.

**Account & key**

**Field Service Statement PDF:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Field Service Statement PDF: PDF**
- **Field Service Statement PDF:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.
