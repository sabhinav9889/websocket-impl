package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisClient struct {
	Client *redis.Client
}

func NewRedis(host, port, username, password string) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Username: username,
		Password: password,
	})
	return &RedisClient{Client: rdb}
}

func (r *RedisClient) SetUserServer(userID, serverID string) {
	r.Client.Set(context.Background(), "user_server:"+userID, serverID, 0)
}

func (r *RedisClient) GetUserServer(userID string) (string, error) {
	return r.Client.Get(context.Background(), "user_server:"+userID).Result()
}

func (r *RedisClient) SetWithTTL(key, value string, ttl time.Duration) {
	r.Client.Set(context.Background(), key, value, ttl)
}

func (r *RedisClient) Exists(key string) bool {
	count, _ := r.Client.Exists(context.Background(), key).Result()
	return count > 0
}

func (r *RedisClient) Subscribe(channelName string, handler func(string)) {
	pubsub := r.Client.Subscribe(context.Background(), channelName)
	defer pubsub.Close()
	for msg := range pubsub.Channel() {
		handler(msg.Payload)
	}
}

func (r *RedisClient) Publish(channel, message string) {
	r.Client.Publish(context.Background(), channel, message)
}

func (r *RedisClient) DeleteUserServer(userID string) {
	r.Client.Del(context.Background(), "user_server:"+userID).Result()
}
