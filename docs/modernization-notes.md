# Modernization notes

What changed between the original example (`terraform-plugin-sdk/v2` v2.35.0, October 2024) and the Go template in `src/go`, and why. Background on the library choice is in [provider-development-options.md](provider-development-options.md).

## Dependencies

| | Example | Template |
| --- | --- | --- |
| Provider library | terraform-plugin-sdk/v2 v2.35.0 | terraform-plugin-framework v1.19.0 |
| Plugin protocol | 5 | 6 |
| Go | 1.22 | 1.25 (required by framework v1.19) |
| Validation | none | terraform-plugin-framework-validators v0.19.0 |
| Logging | standard `log` | terraform-plugin-log v0.11.0 (`tflog`) |
| Testing | none | terraform-plugin-testing v1.16.0 |

These were the latest releases on 2026-10-13.

## Terminology

| SDKv2 | Framework |
| --- | --- |
| `plugin.Serve` with `ProviderFunc` | `providerserver.Serve` with a provider factory and a registry address |
| `*schema.Provider` with `ResourcesMap` | A type implementing `provider.Provider`, with a `Resources` method |
| `*schema.Resource` with function fields | A type implementing `resource.Resource` |
| `schema.TypeInt`, `schema.TypeList` | `schema.Int64Attribute`, `schema.ListAttribute` |
| `d.Get("name").(int)` | A model struct with `tfsdk` tags and `types.Int64` fields |
| `d.SetId("")` to delete | Returning from `Delete` without error diagnostics |
| `return fmt.Errorf(...)` | `resp.Diagnostics.AddError(summary, detail)` |
| `m interface{}` meta argument | `Configure` receives the typed client from the provider |

## Behavior carried over

- One resource, renamed from `example_server` to `template_server`, with `uuid_count`, `uuids`, `creation_delay`, `update_delay`, and `deletion_delay`.
- The resource ID is a UUID fetched from the API. The `uuids` list is fetched in a second call.
- UUIDs come from `https://www.uuidtools.com/api/generate/v1/count/{n}` by default.
- `Read` is a no-op because the UUIDs exist only in state.

## Behavior changed

| Change | Reason |
| --- | --- |
| Delay defaults are `0` instead of `15` seconds | A template should not be slow unless asked. Set the delays explicitly to simulate a slow API. |
| UUIDs are regenerated only when `uuid_count` changes | The example's comment stated this intent, but its `Update` regenerated them on every update, including a change to a delay. `ModifyPlan` now keeps the list stable and shows the real diff at plan time. |
| `uuid_count` is validated to 1 through 100 | The API rejects anything else with HTTP 400. The error now appears at `terraform validate`. |
| Delays must be zero or greater | A negative delay was silently accepted. |
| Delays stop when the operation is cancelled | `time.Sleep` ignored Ctrl+C. The wait now selects on the request context. |
| The API endpoint is configurable | New `endpoint` provider attribute and `TEMPLATE_ENDPOINT` variable. This is what lets the tests run against a local fake API. |
| Non-200 responses are errors | The example tried to parse an error body as a UUID list, which produced a confusing JSON error. |
| The number of returned UUIDs is checked | A short response would have been stored as-is. |
| HTTP timeout is 30 seconds, tied to the request context | The example used a fixed 5 seconds on `context.Background()`, detached from Terraform's cancellation. |

## Code fixes

- `ioutil.ReadAll` is deprecated since Go 1.16. Replaced with `io.ReadAll` behind a 1 MiB limit.
- The example used the `Create`, `Read`, `Update`, and `Delete` fields of `schema.Resource`, which SDKv2 deprecated in favor of the context-aware variants. The framework passes a context everywhere.
- `fmt.Errorf("...: %w", err)` was called with a nil `err` when the API returned an empty list.
- The API client moved to its own package, `internal/uuidclient`, with unit tests.
- Code moved from the module root into `internal/provider`.
- The binary no longer lives in the source tree. The example had a 23 MB `.exe` beside the code.

## Added

- Unit tests for the client and acceptance tests for the resource, both hermetic.
- Schema descriptions, and registry documentation generated from them by `tfplugindocs`.
- A `-debug` flag for attaching a debugger.
- Release configuration for GoReleaser and the Terraform Registry manifest.
- A GitHub Actions workflow that runs the tests.

## Not included

- **Import.** The UUIDs cannot be looked up from the API, so there is nothing to import. A resource backed by a real API should implement `resource.ResourceWithImportState`.
- **Data sources, functions, ephemeral resources, actions.** The example had none. The provider's `DataSources` method is the place to start.
- **Per-resource timeouts.** `terraform-plugin-framework-timeouts` adds a `timeouts` block if operations need configurable limits.
