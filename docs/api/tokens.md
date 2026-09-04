# Personal access tokens

Users can create personal access tokens with a name and optional expiration. Administrators can also manage tokens across accounts.

The plaintext secret is shown only when the token is issued. Lore stores a hashed representation for later authentication and records metadata such as creation time, creator, optional expiration, and last use.

Send a token as a bearer credential:

```http
Authorization: Bearer <token>
```

Malformed or rejected explicit bearer credentials are treated as invalid credentials rather than as an anonymous request.

Tokens authenticate API/media routes that permit bearer authentication; role authorization still applies to the resolved Lore user.
