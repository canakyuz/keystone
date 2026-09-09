# 0006. Önbellek iki katmanlı ve singleflight korumalı

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

Tenant kimliğinden şema adına çözümleme her istekte çalışıyor. Sistemin en
sıcak yolu bu.

## Korunması gereken kurallar

- Redis erişilemediğinde servis çalışmaya devam etmeli.
- Var olmayan tenant kimlikleri veritabanını yormamalı.
- Aynı anahtara gelen eşzamanlı ıskalar tek yüklemeye inmeli.

## Değerlendirilen yaklaşımlar

### A. Yalnızca Redis (önceki hali)

Zayıf tarafları, ölçüldüğünde dördü birden çıktı:

- Soğuk bir anahtara aynı anda gelen N istek N veritabanı sorgusu yapıyordu.
  Önbellek, en çok işe yaraması gereken anda korumuyordu.
- Var olmayan kimlikler önbelleklenmiyordu. Rastgele kimliklerle yapılan
  istek seli her seferinde veritabanına iniyordu.
- Önbellek doldurma her istekte yeni bir goroutine açıyordu, panic recovery
  yoktu.
- Redis tek hata noktasıydı.

### B. Yalnızca süreç içi önbellek

Basit ama replikalar arası paylaşım yok. Her replika ayrı ısınır, veritabanı
yükü replika sayısıyla çarpılır.

### C. İki katman: süreç içi artı Redis

## Karar

C, singleflight ve negatif önbellekleme ile.

## Gerekçe

L1 katmanı Redis'i tek hata noktası olmaktan çıkarıyor. Redis düştüğünde
servis L1 ve veritabanı ile çalışmaya devam ediyor.

`singleflight` yığılmayı çözüyor. Aynı anahtar için aynı anda gelen istekler
tek yüklemeye indirgeniyor; test yüz eşzamanlı isteğin tek sorgu yaptığını
doğruluyor.

Negatif önbellekleme ucuz bir yük yükseltme vektörünü kapatıyor.

TTL'e jitter ekleniyor. Aynı anda oluşturulan anahtarlar aynı anda düşerse
sona erme anında toplu bir ıska dalgası oluşur.

## Sonuçları

- Süreçler arası tutarsızlık penceresi var. Bir kaydı geçersiz kılmak yalnızca
  o süreçte anlık etki eder; diğer replikalar L1 TTL'i (30 saniye) dolana
  kadar eski değeri görebilir.
- Bu, tenant şeması için kabul edilebilir çünkü şema yalnızca onboarding
  sırasında değişir. Sık değişen veriler için kabul edilemez.

## Bu karar ne zaman yanlış hale gelir

- Önbelleklenen veri sık değişirse. 30 saniyelik tutarsızlık penceresi
  kabul edilemez hale gelir; o noktada pub/sub ile geçersiz kılma yayını
  gerekir.
- Bellek baskısı oluşursa. L1 üst sınırı 10.000 kayıt; tenant sayısı bunu
  aşarsa tahliye sıklaşır ve L1 faydası azalır.

## Ölçüm notu

Önceki uygulamanın yorumunda "%98-99 isabet oranı" yazıyordu, ölçülmemişti.
`Stats()` eklendi ama henüz metriklere bağlanmadı.
