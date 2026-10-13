# Options for writing a custom Terraform provider

Research date: 2026-10-13. Version numbers and dates come from the Go module proxy; policy statements come from HashiCorp's documentation, linked under [Sources](#sources).

## The question

The example this template is based on was written on 2024-10-31 against `terraform-plugin-sdk/v2` v2.35.0, on the understanding that this was the only way to write a custom provider at the time. Is that accurate?

## Short answer

It depends on which "only option" is meant.

| Claim | Verdict |
| --- | --- |
| Go was the only supported language for providers | **True**, in 2024 and today. |
| The SDKv2 library was the only way to write one | **Not accurate.** The Plugin Framework had been generally available for almost two years and was already the recommended choice. |

## Libraries: SDKv2 was not the only option

| Date | Event |
| --- | --- |
| 2022-12-13 | `terraform-plugin-framework` v1.0.0 tagged. HashiCorp announced general availability the next day. |
| 2024-09-18 | Framework v1.12.0 released. |
| 2024-10-31 | Framework v1.13.0 and SDKv2 v2.35.0 released, the same day the example was written. |
| 2026-03-10 | Framework v1.19.0, the latest at the time of this research. |

So when the example was written there were three ways to build a provider in Go:

1. **Terraform Plugin Framework**. Recommended for new providers since its GA in December 2022.
2. **Terraform Plugin SDKv2**. What the example used. Still supported, but HashiCorp describes it as a way to maintain existing providers on protocol version 5 and says it has "stopped most feature development" on it.
3. **terraform-plugin-go**. The low-level protocol bindings that both of the above are built on. Usable directly, but you implement the type system and schema handling yourself.

SDKv2 was a reasonable thing to land on in 2024, because most tutorials, blog posts, and existing providers still used it. It was the most visible option, not the only one.

### Why the framework matters now

HashiCorp's current guidance is to use the framework for new providers and to migrate existing ones when possible. Newer Terraform features are only available through it:

- Provider-defined functions
- Ephemeral resources
- Actions
- List resources
- Dynamic attribute types
- Nested attributes, which need protocol version 6

It also separates configuration, plan, and state, and distinguishes null from unknown values. SDKv2 merges all of that into a single `schema.ResourceData`.

### Protocol versions

| Protocol | Terraform CLI | Implemented by |
| --- | --- | --- |
| 5 | 0.12 and later | SDKv2, framework, terraform-plugin-go |
| 6 | 1.0 and later | framework, terraform-plugin-go |

This template serves protocol version 6, so it needs Terraform 1.0 or later.

## Languages: Go is the only supported one

HashiCorp's position, unchanged across every version of the page checked:

- It is "technically possible to write providers in languages other than Go."
- HashiCorp will not document how to do it and will not provide support for it.
- It "may reconsider this policy" once the provider interface is more stable.

### How a non-Go provider would work

A provider is a standalone executable that Terraform starts as a child process and talks to over gRPC on a local socket. Nothing in that requires Go. A provider in another language has to:

1. Implement the go-plugin handshake: print a line of the form `CORE-PROTOCOL-VERSION|APP-PROTOCOL-VERSION|NETWORK-TYPE|NETWORK-ADDR|PROTOCOL` to stdout, for example `1|6|tcp|127.0.0.1:1234|grpc`.
2. Register the gRPC health checking service.
3. Implement the provider service generated from `tfplugin6.proto`, which HashiCorp publishes in the `hashicorp/terraform` repository under `docs/plugin-protocol`.
4. Reimplement what the Go libraries provide: Terraform's type system, the msgpack and JSON encodings of dynamic values, schema handling, and plan logic.

Step 4 is the expensive one. It is the work the framework does for you in Go.

### Community SDKs

These were found by searching on 2026-10-13. None is affiliated with HashiCorp. Star counts are as shown on GitHub that day.

| Language | Project | Notes |
| --- | --- | --- |
| C# | [SamuelFisher/TerraformPluginDotNet](https://github.com/SamuelFisher/TerraformPluginDotNet) | 52 stars, 49 commits, no tagged releases. Covers provider configuration, resource CRUD, import, schema upgrade, and data sources. Its instructions target Terraform 0.13.3 and .NET 7. |
| Python | [hfern/tf](https://github.com/hfern/tf) | 13 stars. On PyPI as `tf`. Targets Terraform and OpenTofu. |
| Rust | [aneoconsulting/tf-provider](https://github.com/aneoconsulting/tf-provider) | 20 stars, 198 commits. Targets Terraform and OpenTofu. |
| Java | None found | Two searches turned up no Java SDK. That is not proof none exists. |

Not verified for any of these: last commit date, which protocol version they implement, and whether they work with current Terraform releases. Check those before relying on one.

### C# and Java for this template

Both can be used in principle, and neither is a good default.

- **C#** has one small community SDK. A provider built on it would trail the protocol, with no support for functions, ephemeral resources, or actions unless the SDK adds them.
- **Java** has no SDK that this research found, so it means implementing the protocol from the `.proto` files. Distribution is also awkward. Terraform expects a single executable per platform, so a Java provider needs a GraalVM native image or a launcher script plus a bundled runtime. This point is an inference from how providers are installed, not something tested here.
- Neither has a counterpart to `terraform-plugin-testing` or `terraform-plugin-docs`.

Recommendation: keep Go as the reference implementation. If a team's API logic lives in C# or Java, the lower-risk design is a thin Go provider that calls that code over HTTP or gRPC, which is the shape this template already has.

## Sources

- [Terraform Plugin Framework is now generally available](https://www.hashicorp.com/blog/terraform-plugin-framework-is-now-generally-available), HashiCorp blog, 2022-12-14
- [Plugin Framework benefits](https://developer.hashicorp.com/terraform/plugin/framework-benefits)
- [Terraform Plugin SDKv2](https://developer.hashicorp.com/terraform/plugin/sdkv2)
- [Migrating from SDKv2 to the framework](https://developer.hashicorp.com/terraform/plugin/framework/migrating)
- [Terraform plugin protocol](https://developer.hashicorp.com/terraform/plugin/terraform-plugin-protocol)
- [Writing non-Go providers](https://developer.hashicorp.com/terraform/plugin/sdkv2/best-practices/other-languages)
- [Writing plugins without Go](https://github.com/hashicorp/go-plugin/blob/main/docs/guide-plugin-write-non-go.md), go-plugin
- [Plugin protocol definitions](https://github.com/hashicorp/terraform/blob/main/docs/plugin-protocol/README.md), hashicorp/terraform
- [terraform-provider-scaffolding-framework](https://github.com/hashicorp/terraform-provider-scaffolding-framework), HashiCorp's own template
