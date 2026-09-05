package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript, token bucket'i tek bir atomik adimda gunceller.
//
// NEDEN LUA: oku, hesapla, yaz dizisi uc ayri Redis komutuyla yapilirsa,
// iki replika ayni anda okuyup ayni tokeni harcayabilir. Redis, Lua betigini
// tek bir islem olarak calistirdigi icin yaris kosulu tasarim geregi olusmaz.
// WATCH/MULTI ile optimistic locking de mumkundu, ama cakismada yeniden deneme
// gerektirir ve yuksek yukte tam da limitin devreye girdigi anda maliyeti artar.
//
// KEYS[1] : kova anahtari
// ARGV[1] : kapasite (burst)
// ARGV[2] : saniyedeki dolum hizi
// ARGV[3] : simdi (milisaniye)
// ARGV[4] : anahtarin yasam suresi (saniye)
//
// Donus: {izin(0/1), kalan_token, yeniden_deneme_ms}
const tokenBucketScript = `
local key      = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local ttl      = tonumber(ARGV[4])

local state     = redis.call('HMGET', key, 'tokens', 'ts')
local tokens    = tonumber(state[1])
local last      = tonumber(state[2])

if tokens == nil then
  tokens = capacity
  last   = now
end

-- Gecen sureye gore dolum. Saat geri giderse negatif sureyi yok say.
local elapsed = math.max(0, now - last) / 1000.0
tokens = math.min(capacity, tokens + elapsed * rate)

local allowed = 0
local retry   = 0

if tokens >= 1 then
  allowed = 1
  tokens  = tokens - 1
else
  if rate > 0 then
    retry = math.ceil(((1 - tokens) / rate) * 1000)
  else
    retry = ttl * 1000
  end
end

redis.call('HSET', key, 'tokens', tokens, 'ts', now)
redis.call('EXPIRE', key, ttl)

return {allowed, math.floor(tokens), retry}
`

// Redis, replikalar arasinda paylasilan token bucket limitleyicisidir.
type Redis struct {
	client *redis.Client
	script *redis.Script
	prefix string

	// FailOpen, Redis erisilemedigi zaman isteklerin gecirilip
	// gecirilmeyecegini belirler.
	//
	// Varsayilan acik (gecir). Gerekce: limitleyici bir kullanilabilirlik
	// araci degil, kotuye kullanim frenidir. Redis dustugunde tum trafigi
	// reddetmek, onlemeye calistigi kesintiyi kendi eliyle yaratir.
	// Kimlik dogrulama gibi kotuye kullanimin pahali oldugu uclarda bu
	// bilincli olarak kapatilabilir.
	FailOpen bool
}

// NewRedis, paylasimli limitleyici olusturur.
func NewRedis(client *redis.Client, keyPrefix string) *Redis {
	return &Redis{
		client:   client,
		script:   redis.NewScript(tokenBucketScript),
		prefix:   keyPrefix,
		FailOpen: true,
	}
}

// Allow, anahtar icin bir token tuketmeyi dener.
//
// Karmasiklik: tek Redis gidis donusu, O(1) sunucu tarafi is.
func (r *Redis) Allow(ctx context.Context, key string, quota Quota) (Result, error) {
	if quota.Burst <= 0 {
		return Result{Allowed: false}, nil
	}

	ttl := bucketTTL(quota)

	raw, err := r.script.Run(ctx, r.client,
		[]string{r.prefix + key},
		quota.Burst, quota.Rate, time.Now().UnixMilli(), int(ttl.Seconds()),
	).Slice()
	if err != nil {
		return r.onFailure(quota), fmt.Errorf("rate limit sorgusu başarısız: %w", err)
	}

	return parseResult(raw, quota)
}

// bucketTTL, kovanin ne kadar yasayacagini hesaplar.
//
// Bos bir kovanin tamamen dolmasi icin gereken sureden kisa olmamalidir;
// aksi halde anahtar erken silinir ve istemci kotasini sifirdan kazanir.
func bucketTTL(quota Quota) time.Duration {
	if quota.Rate <= 0 {
		return time.Hour
	}

	fill := time.Duration(float64(quota.Burst)/quota.Rate) * time.Second

	return max(2*fill, time.Minute)
}

// onFailure, Redis erisilemediginde uygulanacak davranisi uretir.
func (r *Redis) onFailure(quota Quota) Result {
	return Result{
		Allowed:    r.FailOpen,
		Limit:      quota.Burst,
		Remaining:  0,
		RetryAfter: time.Second,
	}
}

// parseResult, Lua betiginin donusunu Result'a cevirir.
func parseResult(raw []any, quota Quota) (Result, error) {
	if len(raw) != 3 {
		return Result{}, fmt.Errorf("beklenmeyen rate limit yanıtı: %d alan", len(raw))
	}

	allowed, _ := raw[0].(int64)
	remaining, _ := raw[1].(int64)
	retryMS, _ := raw[2].(int64)

	return Result{
		Allowed:    allowed == 1,
		Limit:      quota.Burst,
		Remaining:  int(remaining),
		RetryAfter: time.Duration(retryMS) * time.Millisecond,
	}, nil
}
