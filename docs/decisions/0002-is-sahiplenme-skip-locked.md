# 0002. İş sahiplenme FOR UPDATE SKIP LOCKED ile yapılır

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

Birden fazla worker aynı kuyruktan iş alıyor. Aynı işin iki worker tarafından
yürütülmesi, şema oluşturma gibi adımların iki kez çalışması demek.

## Korunması gereken kurallar

- Aynı işi iki worker aynı anda sahiplenemez.
- Bir worker'ın meşgul olması diğerini bekletmemelidir.
- Sahiplenme, kontrol ile yazma arasında yarışa açık olmamalıdır.

## Değerlendirilen yaklaşımlar

### A. Uygulama tarafında kilit

Redis veya bellek içi bir kilit alınır, sonra iş güncellenir.

Zayıf tarafı: kilit ile veritabanı yazımı iki ayrı sistemde. Kilit alındıktan
sonra yazma başarısız olursa kilit yetim kalır. Ayrıca kilidin kendisi bir
kullanılabilirlik bağımlılığı ekler.

### B. SELECT sonra UPDATE

Bekleyen iş okunur, sonra sahiplenme yazılır.

Zayıf tarafı: iki adım arasında başka bir worker aynı işi okuyabilir. Klasik
kontrol-sonra-yaz yarışı. Serializable izolasyon seviyesi bunu çözer ama
çakışmada yeniden deneme gerektirir ve tam da yük arttığında maliyeti artar.

### C. FOR UPDATE SKIP LOCKED

Tek ifadede satır kilitlenir, kilitli satırlar atlanır, sahiplenme yazılır.

## Karar

C.

## Gerekçe

Sahiplenme atomikliği veritabanının kendi kilit mekanizmasına düşüyor. İkinci
bir kilit katmanı, ikinci bir hata kaynağı demek.

`SKIP LOCKED` kısmı önemli. Onsuz, ikinci worker birincinin işlemini
bitirmesini bekler ve kuyruk fiilen tek işlemciye düşer. Atlayarak ilerlemek,
N worker'ın N farklı işi paralel almasını sağlar.

Kısmi indeks (`WHERE status IN ('pending','running')`) sayesinde tamamlanmış
işler taranmıyor, dolayısıyla tablo büyüdükçe sahiplenme maliyeti sabit kalıyor.

## Sonuçları

- Kuyruk PostgreSQL'e bağımlı. Ayrı bir kuyruk sistemine geçmek bu sorguyu
  baştan yazmayı gerektirir.
- Sıralama `next_attempt_at` üzerinden. Öncelik veya adil zamanlama eklemek
  sorguyu ve indeksi değiştirmeyi gerektirir.

## Bu karar ne zaman yanlış hale gelir

- İş hacmi veritabanının taşıyabileceğinin üstüne çıkarsa. Sahiplenme sorgusu
  her worker döngüsünde çalışır; yüksek worker sayısında kilit çekişmesi
  darboğaza dönüşebilir.
- İşler saniyeler değil milisaniyeler sürüyorsa. O ölçekte veritabanı gidiş
  dönüşü işin kendisinden pahalı olur.
- Adil zamanlama gerekirse. Tek bir tenant'ın kuyruğu doldurması küçük
  tenant'ları geciktiriyorsa, basit sıralama yetmez.

## Ölçüm notu

Bu sorgunun maliyeti varsayılmadı, ölçülmeli. Sorgu planı, indeks kullanımı ve
kilit çekişmesi ayrı ayrı bakılmalı. Henüz yapılmadı.
