# Responsible Use and Data Handling

PromptCache is self-hosted semantic-caching middleware. This page highlights deployment choices that matter when prompts or responses contain personal, confidential, regulated, user-specific, or multi-tenant data.

This is operational guidance, not legal advice, and it does not make a deployment compliant with any law, regulation, or provider policy.

## Cached data

PromptCache may persist prompts, model responses, embeddings, cache metadata, and historical prompt/response pairs supplied through cache warming. Treat the cache as potentially sensitive storage.

The cache has a configurable TTL (24 hours by default), but a TTL alone is not a complete retention policy. Consider backups, logs, exports, and any surrounding systems when you define deletion and retention requirements.

## Security boundaries

**Semantic similarity is not an authorization mechanism.**

Do not let a response created for one user, tenant, or authorization scope reach another solely because their prompts are similar. Application-level authorization must run independently of cache matching.

PromptCache does not currently turn an arbitrary client-supplied identifier into a trusted authorization boundary. If isolated users or tenants share one deployment, partition the cache using a mechanism tied to a trusted identity, or use separate deployments. A namespace or partition key is useful only when callers cannot use it to bypass your access-control model.

Semantic matching is probabilistic. Thresholds and verifier models reduce incorrect matches but cannot guarantee that two requests are interchangeable or that a cached answer is still fresh. Bypass caching, or add application-level validation, where a wrong or stale answer would have unacceptable consequences.

## Deployment

For non-local deployments:

- Set `API_AUTH_TOKEN` and restrict access to management endpoints.
- Protect provider credentials, the BadgerDB directory, and backups.
- Limit network exposure and review logs/metrics for sensitive data.
- Define which requests may be cached and how long cached data should be retained.
- Keep PromptCache and its dependencies updated.

PromptCache is not itself a security boundary and does not by itself establish GDPR, HIPAA, PCI DSS, or other regulatory compliance.

## Provider terms

PromptCache forwards requests to third-party model and embedding providers chosen by the deployer. Review the current terms, account restrictions, privacy/data-processing terms, and output-use rules for the providers you configure. Those terms can change independently of PromptCache.

## Trademarks and affiliation

PromptCache is an independent open-source project. It is not affiliated with, endorsed by, or sponsored by OpenAI, Anthropic, Mistral AI, or Voyage AI. Product and company names are used only to describe compatibility and provider support; trademarks belong to their respective owners.

## License

PromptCache is distributed under the MIT License and is provided on an "AS IS" basis, without warranty of any kind. See [`LICENSE`](LICENSE) for the complete terms.

See [`SECURITY.md`](SECURITY.md) for vulnerability reporting.
