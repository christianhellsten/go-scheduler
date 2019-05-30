# go-scheduler

Your preferred scheduling solution™

## Requirements

- PostgreSQL for storing schedules
- Redis for receiving and storing jobs
- Sidekiq, or compatible library, for executing jobs
- Go for polling PostgreSQL and publishing jobs to Redis

## Database

```sql
CREATE TABLE schedules (
  id uuid DEFAULT gen_random_uuid() NOT NULL,
  period integer NOT NULL,
  period_unit character varying NOT NULL,
  starting_from timestamp without time zone NOT NULL,
  ending_at timestamp without time zone NOT NULL,
  last_run_at timestamp without time zone,
  lock_version integer DEFAULT 0 NOT NULL,
  created_at timestamp without time zone NOT NULL,
  updated_at timestamp without time zone NOT NULL
);
```

## Running

```bash
$ go get
$ PGUSER=scheduler PGPASSWORD= PGDATABASE=scheduler_development PGSSLMODE=disable go run main.go
```

## TODO

- [ ] Remove go-workers dependency
