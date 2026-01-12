package services

type ShardedPostgres struct {
	resolver *Resolver
	shards   map[int]*PostgresStorage
}

var shardedPG *ShardedPostgres
