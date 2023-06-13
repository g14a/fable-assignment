### Fable Backend Assignment

Add an `.env` file containing the following values:

```dotenv
POSTGRES_HOST=postgres
POSTGRES_USER=postgres
POSTGRES_PASSWORD=uisvh25223c
POSTGRES_DB=fable
APP_HOST=app
```

Simply run `docker compose up app` to set up Postgres and application containers.

Then run `docker compose up test` to run the test script i.e `test/test.go`

### Overview

1. I'm using [golang-migrate](https://github.com/golang-migrate/migrate) to run schema migrations on Postgres.
1. The test script sends requests to the `POST /log` API in the following format
```json
{
    "id": 1,
    "unix_ts": "timestamp-format",
    "user_id": 1,
    "event_name": "login"
}
```
2. A `logs.txt` file gets generated on startup and the application flushes the logs to the db every 30 seconds.

PS: I've tried running the application and sending a massive load of 100k requests per second and my machine broke. 
Load testing took some time. I could only go until 10k requests per sec on my machine finally.