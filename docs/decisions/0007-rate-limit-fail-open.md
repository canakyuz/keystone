# 0007. İstek limiti paylaşımlı ve Redis düştüğünde açık kalır

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

Fiber'ın yerleşik limiter'ı varsayılan olarak süreç içi sayıyordu. Üç
replikada, replika başına yüz istek ayarı gerçekte üç yüz istek demekti.
Ayarlanan değer ile uygulanan değer arasında replika sayısı kadar fark vardı.

Ayrıca anahtar IP'ydi, dolayısıyla limit kiracıya göre değil bağlantıya göre
uygulanıyordu.

## Korunması gereken kurallar

- Ayarlanan limit, replika sayısından bağımsız olarak uygulanmalı.
- Bir kiracının trafiği diğerinin limitini tüketmemeli.
- Sayaç güncellemesi atomik olmalı.

## Değerlendirilen yaklaşımlar

### A. Sabit pencere sayacı

Basit. Zayıf tarafı: pencere sınırında iki kat geçişe izin verir. Bir pencerenin
sonunda ve bir sonrakinin başında limit kadar istek geçebilir.

### B. Kayan pencere kaydı

Doğru ama her istek için zaman damgası saklar. Bellek maliyeti istek hızıyla
büyür.

### C. Token bucket, Redis'te Lua ile

Kapasite kadar patlamaya bilinçli izin verir, sonrasında sabit hıza düşer.
Oku, hesapla, yaz dizisi tek betikte çalışır.

## Karar

C.

## Gerekçe

Lua betiği atomikliği sağlıyor. Üç ayrı Redis komutuyla yapılsaydı iki replika
aynı tokeni harcayabilirdi. `WATCH`/`MULTI` ile optimistic locking de mümkündü,
ama çakışmada yeniden deneme gerektirir ve tam da limitin devreye girdiği anda
maliyeti artar.

Test iki yüz eşzamanlı istekten tam olarak kapasite kadarının geçtiğini
doğruluyor.

Limit kiracı planından türetiliyor. Şemada zaten `plan` kolonu vardı.

## Redis düştüğünde: açık kalır

Bu bilinçli bir karar ve tartışmalı olduğu için ayrıca yazılıyor.

Varsayılan davranış isteği geçirmek. Gerekçe: limitleyici bir kullanılabilirlik
aracı değil, kötüye kullanım freni. Redis düştüğünde tüm trafiği reddetmek,
önlemeye çalıştığı kesintiyi kendi eliyle yaratır.

Karşı argüman geçerli: kötüye kullanımın pahalı olduğu uçlarda, örneğin kimlik
doğrulamada, açık kalmak brute-force'a kapı açar. Bu yüzden davranış
yapılandırılabilir (`FailOpen`), uç bazında kapatılabilir. Henüz hiçbir uçta
kapatılmadı.

## Sonuçları

- Her istek bir Redis gidiş dönüşü ekliyor. Sağlık uçları muaf tutuldu.
- Plan çözümlemesi ayrıca önbellekleniyor, yoksa limitleyicinin kendisi bir
  yük kaynağına dönüşürdü.

## Bu karar ne zaman yanlış hale gelir

- Redis gecikmesi istek gecikmesinde belirgin paya sahip olursa. O noktada
  yerel bir ön filtre artı periyodik senkronizasyon gerekir.
- Kötüye kullanım maliyeti kullanılabilirlik maliyetini geçerse. Fail-open
  varsayılanı tersine çevrilmeli.
