# Database verification — Telegram bot integration API

Covers task group 11 of the change `2026-09-07-add-telegram-bot-api`.

Schema impact under test: one additive column `users.telegram_username`, `varchar(32)`, nullable, no default, no index, no constraint. No other DDL.

## Engines exercised

| Engine | Version reported by the server | Why this version |
|---|---|---|
| SQLite | `glebarez/sqlite` as vendored by the project | The embedded default |
| MySQL | `5.7.44` | Minimum supported version per the project rules — the version most likely to break |
| PostgreSQL | `9.6.24` | Minimum supported version per the project rules |

MySQL and PostgreSQL ran as throwaway containers on non-default ports (13306, 15432), removed after the run.

Binaries under test, both built from a real frontend build so the embed directive resolved:

- `new-api-HEAD.exe` — built from commit `5cd98aefe` in a detached worktree, i.e. the released schema without this change
- `new-api-NEW.exe` — built from the working tree with the change applied

One startup is one migration: the harness starts the binary, polls `/api/status` until it answers — which only happens after startup, including migration, has completed — then stops the process.

## Matrix and results

| Engine | Fresh database | Upgrade from the previous release | Idempotency on restart | Existing data, indexes, constraints preserved |
|---|---|---|---|---|
| SQLite | Pass | Pass | Pass | Pass |
| MySQL 5.7.44 | Pass | Pass | Pass | Pass |
| PostgreSQL 9.6.24 | Pass | Pass | Pass | Pass |

### MySQL 5.7.44

Upgrade path: the previous-release binary created the schema first — 36 tables, and `SHOW COLUMNS FROM users LIKE 'telegram_username'` returned nothing, confirming the correct pre-change baseline. The new binary then started three times against that same database.

- Column after upgrade: `telegram_username`, `varchar(32)`, `Null=YES`, no key, no default.
- Indexes referencing the new column: `0`.
- Total distinct indexes on `users`: `17` before, `17` after. The column added none.
- `SHOW CREATE TABLE users` captured after the second and third startups differed by exactly one token, `AUTO_INCREMENT=2`, caused by the verification row inserted between them. Every column definition and all 17 index definitions were byte-identical.
- A row written before the third startup survived it intact, including its `telegram_username` value.

Fresh path: `SHOW CREATE TABLE users` after the first and second startups was byte-identical.

Note on setup: the first attempt failed because the container's default schema charset was `latin1`, which the application rejects at startup by design. Recreating the schema as `utf8mb4` resolved it. That was a harness defect, not a finding about the change.

### PostgreSQL 9.6.24

Upgrade path: the previous-release binary created 36 tables with 17 indexes on `users` and no `telegram_username`. A row was inserted, then the new binary started twice.

- Column after upgrade: `character varying(32)`, nullable.
- Indexes referencing the new column: `0`.
- Total indexes on `users`: `17` before, `17` after.
- `\d users` captured after each of the two startups was identical.
- The pre-existing row survived with its values intact.

Fresh path: `\d users` after the first and second startups was identical.

### SQLite

Upgrade path: the previous-release binary created a database without the column; the new binary then started three times.

- Column after upgrade: `telegram_username varchar(32)`.
- Indexes referencing the new column: `0`.

Both fresh and upgrade paths showed the column list identical across restarts, and the index set identical once sorted, since `.schema` does not emit indexes in a stable order.

One difference did appear across restarts and was investigated rather than assumed benign: the table definition changes from ``CREATE TABLE `users` `` on the first startup to `CREATE TABLE "users"` on the second, which is the signature of SQLite rebuilding the table, since SQLite implements most schema alteration as a rebuild.

A control run settled the question. The previous-release binary, containing none of this change, was started twice against its own fresh SQLite database and produced the same backtick-to-quote transition. The rebuild is therefore pre-existing behaviour of the project's GORM automatic migration on SQLite and is not introduced by this change. It is recorded here because it is real, it is unrelated, and a future reader comparing SQLite schemas across restarts will otherwise be misled by it.

## Verdict

The change is compatible with SQLite, MySQL 5.7.44, and PostgreSQL 9.6.24, on both a fresh database and a database created by the previous release, and its migration is idempotent across restarts on all three.

## Unresolved

The SQLite table rebuild on every restart is pre-existing and outside this change. It is harmless on small databases but rewrites the whole table each time the process starts, so it is worth a separate look if SQLite is used at scale.
