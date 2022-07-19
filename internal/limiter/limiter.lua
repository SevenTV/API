local key = ARGV[1]
local expire = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local by = tonumber(ARGV[4])

local exists = redis.call("EXISTS", key)

local count = redis.call("INCRBY", key, by)

if exists == 0 then
    redis.call("EXPIRE", key, expire)
    return {count, expire}
end

local ttl = redis.call("TTL", key)

if count > limit then
    return {redis.call("DECRBY", key, by), ttl, 1}
end

return {count, ttl, 0}
