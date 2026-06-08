# apple-business-go

[![Go Reference](https://pkg.go.dev/badge/github.com/hitoshiichikawa/apple-business-go.svg)](https://pkg.go.dev/github.com/hitoshiichikawa/apple-business-go)
[![CI](https://github.com/hitoshiichikawa/apple-business-go/actions/workflows/ci.yml/badge.svg)](https://github.com/hitoshiichikawa/apple-business-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hitoshiichikawa/apple-business-go)](https://goreportcard.com/report/github.com/hitoshiichikawa/apple-business-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An **unofficial Go SDK for Apple Business**. A library for calling the Apple Business / School Manager API
(`api-business.apple.com` / `api-school.apple.com`). It supports OAuth2 (client_credentials + ES256 JWT)
authentication, reading devices / users / MDM servers and more, and **assigning / unassigning devices (writes)**.

> In April 2026, Apple Business Manager, Apple School Manager, and Apple Business Essentials were unified
> into a single platform, **"Apple Business"** (four pillars: device / people / **brand** / support).
> This SDK is structured around that umbrella brand, anticipating future pillars (such as brand management).

> ⚠️ This is an **unofficial SDK**. The API definitions were extracted and verified from public implementations,
> but final confirmation against Apple's official documentation is recommended before production use
> (see `docs/apple-business-api-reference.md`).

---

## Package layout (mapped to the four pillars)

```
apple-business-go/
├── applebusiness/   Core (shared foundation): Client, Config, Credentials, OAuth2/JWT, transport, pagination, errors
├── devices/         Device management: orgDevices, mdmServers, orgDeviceActivities, appleCareCoverage
├── blueprints/      Blueprint management: CRUD + assignment (apps/configurations/devices/users/groups)
├── configurations/  Configuration management: CRUD (CUSTOM_SETTING profiles)
├── apps/            Apps / packages (read-only): apps, packages
├── auditevents/     Audit events (read-only): auditEvents
├── people/          People management: users, userGroups
├── brand/           Brand management (future / Apple Business Connect)
└── support/         Support (future)
```

- The service packages (`devices` / `people`) accept a `*applebusiness.Client` and operate on it.
- Authentication, retries, and pagination are centralized in the `applebusiness` core; the core stays unchanged as pillars are added.

---

## Install

```bash
go get github.com/hitoshiichikawa/apple-business-go/applebusiness
go get github.com/hitoshiichikawa/apple-business-go/devices
```

## Quickstart (read)

```go
import (
    "github.com/hitoshiichikawa/apple-business-go/applebusiness"
    "github.com/hitoshiichikawa/apple-business-go/devices"
)

pem, _ := os.ReadFile("abm_private_key.pem") // EC P-256 (.pem)

c, err := applebusiness.NewClient(applebusiness.Config{
    BaseURL: applebusiness.DefaultBusinessBaseURL, // use DefaultSchoolBaseURL for ASM
    Credentials: applebusiness.Credentials{
        ClientID:   "BUSINESSAPI.xxxxxxxx-....",
        TeamID:     "BUSINESSAPI.xxxxxxxx-....", // typically identical to client_id on AxM
        KeyID:      "xxxxxxxx-....",
        PrivateKey: pem,
        Scope:      "business.api", // auto-detected from client_id when empty
    },
})

devs, err := devices.New(c).List(context.Background(), nil) // pagination handled automatically
for _, d := range devs {
    fmt.Println(d.Attributes.SerialNumber, d.Attributes.Status)
}
```

## Quickstart (write: assign / unassign)

```go
svc := devices.New(c)
act, err := svc.Assign(ctx, serverID, []string{deviceID1, deviceID2})
// to unassign: svc.Unassign(ctx, serverID, deviceIDs)

final, err := svc.PollActivity(ctx, act.ID, 3*time.Second)
fmt.Println(final.Attributes.Status, final.Attributes.SubStatus)
```

Runnable samples live in [`examples/`](./examples) (`list-devices`, `assign-devices`, `smoke-test`, `dump-all`, `write-test`).
For a connectivity check use [`smoke-test`](./examples/smoke-test); to dump every read endpoint use [`dump-all`](./examples/dump-all).

## Quickstart (audit events)

```go
// /v1/auditEvents requires filter[startTimestamp]; use ListRange for a time range
events, _ := auditevents.New(c).ListRange(ctx, time.Now().AddDate(0, 0, -7), time.Now(), nil)
for _, ev := range events {
    a := ev.Attributes // common fields: a.Type, a.ActorID, a.EventDateTime ...
    if a.Type == "DEVICE_ASSIGNED_TO_SERVER" {
        var d auditevents.DeviceAssignedToServer
        _ = a.Payload(&d) // d.SerialNumber, d.TargetServerName
    }
}
```

---

## Authentication

Issue credentials in the Apple Business / School Manager portal (Organization Administrator only).

| Item | Description |
|---|---|
| `client_id` | e.g. `BUSINESSAPI.<uuid>` |
| `key_id` | the `kid` in the JWT header |
| `team_id` | issuer (typically identical to `client_id` on AxM) |
| private key | `.pem` (EC P-256 / for ES256) |

Generate a JWT client assertion signed with ES256 and exchange it at
`POST https://account.apple.com/auth/oauth2/token` with `grant_type=client_credentials`.
The `scope` is `business.api` / `school.api`. Access tokens are valid for one hour.

> [!IMPORTANT]
> **Create one `Client` per credential and reuse it. Do not call `NewClient` on every
> request / API call.**
>
> A `Client` caches its OAuth access token internally via `oauth2.ReuseTokenSource`, but
> only for the lifetime of that single `Client` instance. If you build a new `Client` for
> each API call, **you mint a fresh access token every time**. Under bursty load (e.g. a UI
> that fetches several resources back-to-back), the token endpoint
> `POST https://account.apple.com/auth/oauth2/token` starts returning
> **`429 invalid_request "Too many requests"`**, which then makes the API call itself fail.
>
> Note that this failure surfaces as a `*url.Error`
> (`Get ".../v1/...": applebusiness oauth: token failed (429): ...`) and is **not** an
> `*applebusiness.APIError` — `errors.As(err, &apiErr)` will not match it, so it is reported
> as a transport / unreachable error rather than an Apple API status error.
>
> Mitigations:
> - **Long-lived processes: create the `Client` once and share it** (simplest and safest).
> - With multiple credentials, **cache one `Client` per credential** and reuse it.
> - If you must avoid keeping the decrypted private key resident in memory, do not cache the
>   whole `Client`; instead **cache only the access token and decrypt the key solely on token
>   refresh** (≈ once per hour) via a custom `oauth2.TokenSource` (injectable with
>   `WithHTTPClient`).
> - When multiple requests can miss the cache concurrently, **serialize token issuance with
>   single-flight** so you do not fan out simultaneous token requests.

---

## Supported endpoints

| Package | Methods | API |
|---|---|---|
| `devices` | `List` / `Get` | `/v1/orgDevices`, `/v1/orgDevices/{id}` |
| `devices` | `AssignedServer` / `AppleCareCoverage` | `/v1/orgDevices/{id}/assignedServer`, `.../appleCareCoverage` |
| `devices` | `ListMdmServers` / `MdmServerDevices` | `/v1/mdmServers`, `/v1/mdmServers/{id}/relationships/devices` |
| `devices` | `Assign` / `Unassign` / `GetActivity` / `PollActivity` | `/v1/orgDeviceActivities(/{id})` |
| `people` | `ListUsers` / `GetUser` | `/v1/users`, `/v1/users/{id}` |
| `people` | `ListUserGroups` / `GetUserGroup` / `GroupMembers` | `/v1/userGroups(/{id})(/relationships/users)` |
| `blueprints` | `List` / `Get` / `Create` / `Update` / `Delete` / `AddTo` / `RemoveFrom` / `Replace` / `RelationshipIDs` | `/v1/blueprints(/{id})(/relationships/{rel})` |
| `configurations` | `List` / `Get` / `Create` / `Update` / `Delete` | `/v1/configurations(/{id})` |
| `apps` | `ListApps` / `GetApp` / `ListPackages` / `GetPackage` | `/v1/apps(/{id})`, `/v1/packages(/{id})` |
| `auditevents` | `List` / `ListRange` | `/v1/auditEvents` (requires `filter[startTimestamp]`) |

The primary source for fields and enum values is [`docs/apple-business-api-datatypes.md`](./docs/apple-business-api-datatypes.md);
endpoint details are in [`docs/apple-business-api-reference.md`](./docs/apple-business-api-reference.md).

---

## Status / roadmap

- [x] Core (OAuth2 / retries / pagination)
- [x] `devices` (read + assign/unassign)
- [x] `people` (users / userGroups)
- [x] `apps` (apps / packages) / `auditevents` (auditEvents): implemented (read-only); fields match the official spec
- [x] `blueprints` / `configurations` (built-in device management): implemented (CRUD + assignment); spec confirmed and write paths verified against the live API
- [x] All resource `Attributes`, enum values, and the audit model confirmed against the official DocC ([`docs/apple-business-api-datatypes.md`](./docs/apple-business-api-datatypes.md))
- [x] Functional options (`WithBaseURL`/`WithTokenURL`/`WithMaxRetries`/`WithUserAgent`/`WithHTTPClient`)
- [x] Typed error predicates (`IsNotFound`/`IsRateLimited`/`IsUnauthorized`/`IsForbidden`/`IsConflict`)
- [x] `ListSeq` (lazy paging via Go 1.23 range-over-func)
- [x] Unit tests (`httptest`) for all packages / CHANGELOG / golangci-lint / CI (gofmt, vet, build, test -race, lint)
- [ ] Idempotency review of write retries (avoid duplicate POSTs)
- [ ] `brand` / `support` packages (future; `brand` has no public API spec yet — see [`ROADMAP.md`](./ROADMAP.md))

---

## Development

Go 1.23+ (uses `ListSeq`'s range-over-func).

```bash
go mod tidy
go build ./...
go vet ./...
gofmt -l .              # confirm no diff
golangci-lint run ./...
go test -race ./...
```

---

## Credits

The API definitions and authentication flow were implemented independently, with reference to the following
public projects (no code was copied directly):

- [`micromdm/nanoaxm`](https://github.com/micromdm/nanoaxm) (Go)
- [`neilmartin83/terraform-provider-axm`](https://github.com/neilmartin83/terraform-provider-axm) (Go)
- and others such as `rodchristiansen/asbmutil` (Swift), `EUCTechTopics/PSABM` (PowerShell)

## License

[MIT](./LICENSE). Apple, Apple Business, Apple Business Manager, and Apple School Manager are trademarks of
Apple Inc. This is an unofficial project, not affiliated with Apple.
