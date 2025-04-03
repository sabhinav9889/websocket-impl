package redis

import (
    "github.com/go-redis/redis/v8"
    "context"
    "time"
)

type RedisClient struct {
    client *redis.Client
}

func NewRedis(host, port, username, password string) *RedisClient {
    rdb := redis.NewClient(&redis.Options{
        Addr: host + ":" + port,
		Username: username,
		Password: password,
    })
    return &RedisClient{client: rdb}
}

func (r *RedisClient) SetUserServer(userID, serverID string) {
    r.client.Set(context.Background(), "user_server:"+userID, serverID, 0)
}

func (r *RedisClient) GetUserServer(userID string) string {
    val, _ := r.client.Get(context.Background(), "user_server:"+userID).Result()
    return val
}

func (r *RedisClient) SetWithTTL(key, value string, ttl time.Duration) {
    r.client.Set(context.Background(), key, value, ttl)
}

func (r *RedisClient) Exists(key string) bool {
    count, _ := r.client.Exists(context.Background(), key).Result()
    return count > 0
}

func (r *RedisClient) Subscribe(channel string, handler func(string)) {
    pubsub := r.client.Subscribe(context.Background(), channel)
    for msg := range pubsub.Channel() {
        handler(msg.Payload)
    }
}

func (r *RedisClient) Publish(channel, message string) {
    r.client.Publish(context.Background(), channel, message)
}
