# Security Policy

## Supported Versions

PromptCache is an open-source project provided under the MIT License. Security fixes are generally made on the latest version of the `main` branch. Older releases may not receive backports.

Operators are responsible for tracking updates and assessing whether a newer version is required for their deployment.

## Reporting a Vulnerability

Please do not publish exploit details for a suspected vulnerability before the maintainer has had a reasonable opportunity to investigate it.

Use GitHub's private vulnerability reporting feature for this repository when available. If private reporting is unavailable, open a minimal issue requesting a private contact channel without including sensitive exploit details, credentials, personal data, or production secrets.

A report is most useful when it includes:

- the affected version or commit;
- the deployment assumptions required to reproduce the issue;
- clear reproduction steps;
- the expected and actual behavior;
- the security impact;
- any suggested mitigation, if known.

## Deployment Security

PromptCache is self-hosted middleware. It is not a security or authorization boundary.

Operators are responsible for securing the environment in which PromptCache runs, including network access, host access, API credentials, storage, backups, logs, and observability systems.

In particular:

- Set `API_AUTH_TOKEN` for every non-local deployment.
- Do not expose management endpoints to untrusted networks unless they are appropriately protected.
- Protect provider API keys and other credentials as secrets.
- Protect the BadgerDB data directory and backups using appropriate operating-system, container, and infrastructure controls.
- Restrict access to logs and metrics if prompts, identifiers, or other sensitive metadata may be present.
- Keep PromptCache and its dependencies updated according to your own security policy.

## Cached Data and Privacy

PromptCache may persist prompts, model responses, embeddings, and related metadata. Depending on the application, this data may contain personal, confidential, regulated, or otherwise sensitive information.

Operators are solely responsible for deciding what data may be cached and for configuring retention, deletion, access control, encryption, backups, and other safeguards required by their environment and applicable law.

PromptCache does not by itself establish compliance with GDPR, HIPAA, PCI DSS, or any other legal or regulatory framework.

## Tenant and Security-Context Isolation

Semantic similarity must never be treated as authorization.

A cache entry must not be shared across users, tenants, customers, organizations, authorization scopes, or other security contexts unless the operator has independently determined that those contexts are permitted to share the same cached content.

If an application serves multiple security contexts, the operator should isolate cache namespaces or deployments so that a response created from one context cannot be returned to another context solely because their prompts are semantically similar.

## Semantic-Matching Risk

Semantic matching is probabilistic. Thresholds and verifier models reduce the likelihood of incorrect cache hits but cannot guarantee that every hit has identical meaning, authorization context, freshness requirements, or downstream consequences.

Applications with high-impact, rapidly changing, user-specific, or authorization-sensitive responses should bypass caching or add application-level validation appropriate to the use case.

## Provider Terms

PromptCache forwards requests to third-party model providers selected by the operator. Operators are responsible for complying with the terms, policies, data-processing requirements, and usage restrictions of those providers.

## No Warranty

PromptCache is provided under the MIT License on an "AS IS" basis, without warranty of any kind. See `LICENSE` for the complete license terms.
