package infrastructure

type ShardedPostgres struct {
	resolver *Resolver
	shards   map[int]*PostgresStorage
}
