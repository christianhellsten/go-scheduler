package goscheduler

//
// A simple scheduler written in golang. Schedules are stored in Postgres. Jobs
// are stored in Redis and executed by Sidekiq.
//
import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/jrallison/go-workers"
	"github.com/lib/pq"
	"os"
	"time"
)

var db *sql.DB
var pollInterval time.Duration
var dbTable string
var sidekiqWorker string

func init() {
	dbTable = getEnv("DB_TABLE", "schedules")
	sidekiqWorker = getEnv("SIDEKIQ_WORKER", "JobWorker")
	var err error
	if pollInterval, err = time.ParseDuration(getEnv("POLL_INTERVAL", "1000ms")); err != nil {
		pollInterval, _ = time.ParseDuration("1000ms")
	}
}

// Start ...
func Start() {
	workers.Configure(map[string]string{
		// location of redis instance
		"server": getEnv("REDIS_SERVER", "localhost:6379"),
		// instance of the database
		"database": getEnv("REDIS_DB", "0"),
		// number of connections to keep open with redis
		"pool": getEnv("REDIS_POOL", "30"),
		// unique process id for this instance of workers (for proper recovery of inprogress jobs on crash)
		"process": getEnv("REDIS_PROCESS", "1"),
	})
	var err error
	// XXX: It's critical we set the timezone to UTC!!!
	db, err = sql.Open("postgres", "timezone=UTC")
	if err != nil {
		Fatal(err)
	}
	defer db.Close()
	var timezone string
	err = db.QueryRow("SHOW timezone").Scan(&timezone)
	if err != nil {
		Fatal(err)
	}
	if timezone != "UTC" {
		Fatal(errors.New("db timezone must be UTC"))
	}
	Info("started. db timezone: %s", timezone)
	for {
		time.Sleep(pollInterval)
		pollJobs()
	}
}

func pollJobs() {
	Debug("polling...")
	var id string
	var lockVersion string
	var lastRunAt pq.NullTime
	var deadline time.Time
	sql := `
UPDATE
	%s
SET
	last_run_at=CURRENT_TIMESTAMP
	-- lock_version=lock_version+1
FROM (
	SELECT
		id,
		lock_version,
		last_run_at,
		COALESCE(
		  last_run_at,
			starting_from - CAST(period||' '||period_unit AS Interval)
		) + CAST(period||' '||period_unit AS Interval) AS deadline
	FROM project_schedules
) t
WHERE
	deadline >= starting_from AND
	deadline <= ending_at AND
	deadline < CURRENT_TIMESTAMP
RETURNING t.id, t.lock_version, t.last_run_at, t.deadline;
`
	rows, err := db.Query(fmt.Sprintf(sql, dbTable))
	if err != nil {
		Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&id, &lockVersion, &lastRunAt, &deadline)
		if err != nil {
			Fatal(err)
		}
		// Publish Job to Sidekiq
		jid, err := workers.Enqueue("default", sidekiqWorker, []string{id, lockVersion})
		if err != nil {
			Fatal(err)
		}
		Debug("enqueued jid:", jid, " job_id:", id, " last_run_at:", lastRunAt.Time, " deadline:", deadline, " version:", lockVersion)
	}
	err = rows.Err()
	if err != nil {
		Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}
