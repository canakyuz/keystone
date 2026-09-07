# 0004. Idempotency benzersizlik kısıtıyla zorlanır

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

İstemci `POST /v1/tenants` gönderir, transaction tamamlanır, ama HTTP yanıtı
istemciye ulaşmaz. İstemci aynı isteği tekrar gönderir. İkinci bir tenant
oluşmamalıdır.

## Korunması gereken kurallar

- Aynı kapsamda aynı anahtar, aynı gövdeyle tekrar gelirse aynı operasyonu
  döndürür.
- Aynı anahtar farklı gövdeyle gelirse açık bir çakışma hatası döner.
- Aynı anda gelen iki yinelenen istek iki operasyon yaratamaz.

## Değerlendirilen yaklaşımlar

### A. Önce oku, yoksa yaz

Anahtar var mı diye bakılır, yoksa kayıt eklenir.

Zayıf tarafı: iki istek kontrolü aynı anda geçebilir ve ikisi de yazar.
Tek başına yarış koşulunu engellemez.

### B. Serializable izolasyon

Transaction seviyesi yükseltilir.

Zayıf tarafı: çakışmada yeniden deneme gerektirir. Tam da yükün arttığı anda
maliyeti artar. Ayrıca çağıranın yeniden deneme mantığını taşıması gerekir.

### C. Benzersizlik kısıtı artı ihlali yakalama

`(scope, idempotency_key)` üzerinde unique index. Yazma denenir, kısıt
ihlali yakalanırsa mevcut kayıt okunur.

## Karar

C, önünde ucuz bir okuma ile.

Önce okuma yapılır çünkü sık görülen tekrar durumunu tek sorguyla karşılar.
Aynı anda gelen iki istek bu kontrolü birlikte geçerse, ikincisi INSERT
sırasında kısıtı ihlal eder ve aynı yola düşer.

## Gerekçe

Benzersizliği veritabanı zorluyor. Okuma bir hızlandırma, garanti değil.
Garantinin nerede durduğunu ayırmak önemli: okumayı kaldırsak sistem hâlâ
doğru çalışır, yalnızca yavaşlar.

Farklı gövde için sessizce eski sonucu döndürmek reddedildi. İstemci
göndermediği isteğin işlendiğini sanardı.

## Sonuçları

- İstek gövdesinin normalize edilmiş bir özeti saklanıyor. Normalizasyon
  şu an ham gövdenin SHA-256'sı; alan sırası değişirse farklı özet çıkar.
  Bu bir sınır ve API dokümantasyonunda belirtilmeli.
- Kapsam (`scope`) alanı zorunlu. Bir müşterinin anahtarı diğerinin isteğini
  eşleştirmemeli.

## Bu kararın garanti ETMEDİĞİ şey

Idempotency süresizdir değildir. Anahtarın saklama süresi (varsayılan 24 saat)
dolduktan sonra aynı anahtar yeni bir işlem yaratır. Garanti bu noktada
sessizce değişir ve bu, API dokümantasyonunda yazılmak zorundadır.

## Bu karar ne zaman yanlış hale gelir

- İstemciler saatlerce sonra aynı anahtarla tekrar deniyorsa. Saklama süresi
  uzatılmalı, ama tablo büyümesi ve temizlik işi devreye girer.
- Gövde normalizasyonu yetersiz kalırsa. JSON alan sırası değişen istemciler
  yanlışlıkla çakışma hatası alır. O noktada kanonik JSON serileştirme gerekir.
