# Day 25: PostgreSQL

Start a local database:

    docker run --name mtb-postgres -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=store -p 5432:5432 -d postgres:16

Then run:

    go run index.go

To use a different database, set DATABASE_URL first.