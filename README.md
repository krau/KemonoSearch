# Fast Kemono Creators Search

Sync kemono [creators.txt](https://kemono.su/api/v1/creators.txt) data to sqlite database and provide search api.

## ENVIRONMENTS

```bash
KEMONOSEARCH_DB="creators.db"
KEMONOSEARCH_ADDR=":39808"
```

## ROUTES

| route                  | method | description                        |
| ---------------------- | ------ | ---------------------------------- |
| `/`                    | GET    | Rendered html home page            |
| `/api/creator`         | GET    | Get creators by query param `name` |
| `/api/docs/index.html` | GET    | Swagger UI                         |
