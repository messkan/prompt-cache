# Cache namespaces

PromptCache supports an optional `X-Cache-Namespace` request header for partitioning cached responses, prompts, and embeddings.

```http
X-Cache-Namespace: customer-123
```

Requests with different namespace values do not share cached responses or semantic-search candidates. The namespace value is hashed before it is used in storage keys, so the raw header value is not written into BadgerDB keys.

If the header is omitted, PromptCache uses the existing default cache partition and remains backward compatible with entries created before namespace support.

## Security model

`X-Cache-Namespace` is a **partition key, not an authorization mechanism**.

Do not accept an arbitrary namespace directly from an untrusted end user when namespaces represent security boundaries. A trusted gateway or application layer should derive or validate the namespace from the authenticated tenant, organization, or other authorization context before forwarding the request to PromptCache.

For explicit namespaces, semantic lookup is restricted to embeddings in that namespace. The default partition continues to use the ANN index; explicit namespaces currently use a filtered exact scan to guarantee that ANN candidates cannot cross namespace boundaries.

This favors isolation correctness over ANN performance for namespaced traffic. A future implementation may use one ANN index per namespace or another partition-aware index.
