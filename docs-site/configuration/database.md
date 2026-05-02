# Database

OpenPost uses SQLite by default.

## Default path

The backend code defaults to:

```txt
file:openpost.db?cache=shared&mode=rwc
```

For container deployments, prefer an explicit file path such as:

```txt
/data/db/openpost.db
```

## Operational notes

- Persist the database on durable storage.
- Back up the database together with the media directory.
- Do not keep the database inside ephemeral container layers.
- SQLite is configured for a simple single-node deployment model.
