# Responsible Use and Data Handling

PromptCache is self-hosted semantic-caching middleware. This document describes deployment considerations that operators should evaluate before using it with production, personal, confidential, regulated, multi-user, or multi-tenant data.

This document is operational guidance, not legal advice, and does not make PromptCache compliant with any particular law, regulation, or third-party provider policy.

## Operator Responsibility

The person or organization deploying PromptCache controls the environment, provider credentials, network exposure, cached content, retention practices, and users who can access the deployment. Operators are responsible for determining whether their use of PromptCache is appropriate for their data and for complying with applicable laws, contracts, and provider terms.

## What May Be Stored

Depending on configuration and request flow, PromptCache may persist or process:

- prompts and message content;
- model responses;
- embeddings derived from prompt content;
- cache metadata and identifiers;
- historical prompt/response pairs supplied through cache warming.

These values can contain personal data, confidential business information, source code, credentials accidentally included in prompts, or other sensitive material. Do not assume cached content is anonymous merely because an embedding is also generated.

## Retention and Deletion

Operators should define a retention policy appropriate to their use case. Do not retain cached content indefinitely merely because storage is available.

Where deletion, legal hold, data-subject rights, contractual deletion obligations, or other lifecycle controls apply, operators should ensure that their deployment and surrounding systems provide the required controls. This includes considering primary storage, replicas, backups, exported data, logs, and observability systems.

The existence of a cache-clear or deletion endpoint does not by itself satisfy any particular legal or regulatory requirement.

## Multi-Tenant and Multi-User Deployments

Semantic similarity is not an authorization mechanism.

Do not allow a response generated within one security context to be served to another security context solely because the prompts are semantically similar. Security contexts may include different:

- users;
- tenants or customers;
- organizations;
- roles or authorization scopes;
- data classifications;
- geographic or regulatory boundaries;
- provider accounts or contractual contexts.

For deployments containing data that must remain isolated, use separate cache namespaces or separate PromptCache deployments, or implement an equivalent isolation mechanism before performing semantic cache lookup.

A similarity score, including a high similarity score, must never override application-level authorization.

## Semantic Correctness and Freshness

Semantic caching is probabilistic. Similarity thresholds and verifier models can reduce incorrect matches, but they cannot guarantee that two requests are interchangeable.

Caching may be inappropriate for responses that are:

- user-specific or permission-sensitive;
- rapidly changing or time-sensitive;
- financial, medical, legal, safety-critical, or otherwise high impact;
- dependent on hidden application state;
- required to be freshly generated;
- subject to provider or contractual restrictions on reuse.

Operators should bypass caching or add application-level validation where the consequences of an incorrect or stale cache hit are unacceptable.

## Provider Accounts and Terms

PromptCache integrates with third-party model and embedding providers. Provider names are used descriptively to identify compatibility and integrations.

Operators are responsible for reviewing and complying with the current terms, acceptable-use policies, privacy/data-processing terms, licensing conditions, and account restrictions of every provider they configure. Provider terms can change independently of PromptCache.

Avoid sharing cached content between unrelated provider accounts, customers, or contractual contexts unless you have independently determined that doing so is permitted and appropriate.

## Deployment Checklist

Before production use, operators should at minimum consider:

- setting `API_AUTH_TOKEN` and restricting management endpoints;
- limiting network exposure;
- protecting provider credentials;
- protecting the BadgerDB directory and backups;
- defining cache retention and deletion procedures;
- preventing cache sharing across unauthorized security contexts;
- deciding which requests must bypass caching;
- reviewing logs and metrics for sensitive data;
- maintaining dependency and PromptCache updates;
- reviewing applicable provider terms and data-processing obligations.

See `SECURITY.md` for vulnerability reporting and additional security guidance.

## Trademarks and Affiliation

PromptCache is an independent open-source project. It is not affiliated with, endorsed by, or sponsored by OpenAI, Anthropic, Mistral AI, or Voyage AI.

OpenAI, Anthropic, Claude, Mistral AI, Voyage AI, and other product or company names are trademarks or registered trademarks of their respective owners. Their names are used only to describe interoperability or provider support.

## License and Warranty

PromptCache is distributed under the MIT License and is provided on an "AS IS" basis, without warranty of any kind. See `LICENSE` for the complete terms.
