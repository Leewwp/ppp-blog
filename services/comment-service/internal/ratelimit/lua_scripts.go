package ratelimit

// slidingWindowScript 使用 Redis Sorted Set 实现滑动窗口限流。
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local window_start = now - window * 1000
redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
local current = redis.call('ZCARD', key)
if current < limit then
    redis.call('ZADD', key, now, now .. ':' .. math.random(1000000))
    redis.call('PEXPIRE', key, window * 1000)
    return {1, limit - current - 1}
else
    redis.call('PEXPIRE', key, window * 1000)
    return {0, 0}
end
`
