# Mimari Karar Kayıtları

Her kayıt bir soruyu yanıtlar: bu tasarım neden böyle, alternatifi neydi, ve
hangi koşulda yanlış hale gelir.

Son soru kasıtlı. Bir kararın ne zaman geçersizleşeceğini yazmak, o kararı
gerçekten anladığının kanıtı. Sonsuza kadar doğru olduğunu iddia eden karar
kaydı, karar kaydı değil savunma yazısıdır.

| No | Karar | Not |
|---|---|---|
| [0001](0001-worker-veritabanina-dogrudan-erisir.md) | Worker veritabanına doğrudan erişir | Hedef mimariden bilinçli sapma |
| [0002](0002-is-sahiplenme-skip-locked.md) | İş sahiplenme `FOR UPDATE SKIP LOCKED` ile | |
| [0003](0003-lease-ve-fencing.md) | Sahiplenme lease ve fencing token ile korunur | |
| [0004](0004-idempotency-benzersizlik-kisiti.md) | Idempotency benzersizlik kısıtıyla zorlanır | |
| [0005](0005-schema-per-tenant.md) | Schema-per-tenant, RLS ile birlikte | |
| [0006](0006-onbellek-iki-katmanli.md) | Önbellek iki katmanlı ve singleflight korumalı | |
| [0007](0007-rate-limit-fail-open.md) | İstek limiti paylaşımlı, Redis düşünce açık kalır | Tartışmalı, gerekçesi yazılı |

## Bu kayıtların okunma sırası

Sistemi ilk kez inceleyen biri için: 0005, 0003, 0002, sonra kalanlar.

0005 tenant izolasyonunun ne anlama geldiğini ve neyi kapsamadığını çiziyor.
0003 arızadan toparlanmanın temelini kuruyor. 0002 onun altındaki eşzamanlılık
mekanizmasını anlatıyor.

## Kayıtlarda tekrar eden bir tema

Her kaydın bir "garanti ETMEDİĞİ şey" bölümü var. Bunlar dolgu değil.

- Fencing metadata'yı korur, harici yan etkiyi engellemez.
- Şema ayrımı mantıksal ayrım sağlar, kaynak izolasyonu sağlamaz.
- Idempotency garantisi süresiz değildir.
- İki katmanlı önbellekte süreçler arası tutarlılık anlık değildir.

Bir sistemin neyi garanti etmediğini bilmek, ne yaptığını bilmekten daha zor
ve daha ayırt edici.

## Ayrıca

Sistemin verdiği sözlerin kod ve test karşılıkları için
[../INVARIANTS.md](../INVARIANTS.md) dosyasına bakın.
