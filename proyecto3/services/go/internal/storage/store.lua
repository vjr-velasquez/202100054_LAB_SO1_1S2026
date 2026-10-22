-- One hash and one final HSET: no prior write can survive a validation error.
local key = KEYS[1]
local kind = redis.call('TYPE', key).ok
if kind ~= 'none' and kind ~= 'hash' then return redis.error_reply('CORRUPT wrong key type') end
local previous = redis.call('HGET', key, 'event:' .. ARGV[1])
if previous then
 if previous == ARGV[2] then return 0 else return -1 end
end
local fields = {'count', 'country:' .. ARGV[3] .. ':count', 'planes:' .. ARGV[4], 'ships:' .. ARGV[5]}
local update = {'event:' .. ARGV[1], ARGV[2]}
for _, field in ipairs(fields) do
 local raw = redis.call('HGET', key, field)
 if raw and not string.match(raw, '^%d+$') then return redis.error_reply('CORRUPT invalid counter') end
 local count = tonumber(raw or '0')
 if not count or count >= 9007199254740990 then return redis.error_reply('CORRUPT counter overflow') end
 table.insert(update, field)
 table.insert(update, string.format('%.0f', count + 1))
end
local latestField = 'country:' .. ARGV[3] .. ':latest-order'
local latest = redis.call('HGET', key, latestField)
if not latest or ARGV[6] > latest then
 table.insert(update, latestField)
 table.insert(update, ARGV[6])
 table.insert(update, 'country:' .. ARGV[3] .. ':latest')
 table.insert(update, ARGV[2])
end
redis.call('HSET', key, unpack(update))
return 1
