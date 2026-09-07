# 0003. Sahiplenme lease ve fencing token ile korunur

**Durum:** Kabul edildi, 2026-09-07

## Bağlam

Bir worker işi aldıktan sonra süreç kapanabilir. Kapanma haber verilmez:
sunucu çöker, container öldürülür, ağ bölünür.

İş sahipli görünürken worker ölmüşse, iş kimsenin yürütmediği bir durumda
kalır.

## Korunması gereken kurallar

- Süreç iş ortasında kapanırsa iş yeniden devralınabilir olmalıdır.
- Yalnızca güncel sahip sonuç kaydedebilir.
- Devralınan bir işin eski sahibi, sonucu bozamamalıdır.

## Değerlendirilen yaklaşımlar

### A. Kalıcı "işleniyor" bayrağı

İş alındığında `status = 'processing'` yazılır, bitince değişir.

Zayıf tarafı: worker ölürse bayrak sonsuza kadar kalır. İş hiçbir zaman
devralınmaz. Kurtarma için manuel müdahale veya "şu kadar süredir processing
olanları sıfırla" gibi bir temizlik işi gerekir. İkincisi zaten lease'in
kötü uygulanmış hali.

### B. Yalnızca lease

Sahiplenme süreli yazılır. Süre dolunca iş devralınabilir.

Zayıf tarafı: eski worker geri dönüp sonuç bildirebilir. Kendi adını bildiği
için `lease_owner` kontrolünü geçer. Güncel sahibin işini bozar.

### C. Lease artı fencing token

Sahiplenme süreli, ve her sahiplenmede artan bir sayaç tutulur. Sonuç bildirimi
güncel sayaç değeriyle yapılmak zorundadır.

## Karar

C.

## Gerekçe

B tek başına yetersiz ve bunun nedeni ince. Eski worker'ın kimliği değişmiyor;
kimlik kontrolü onu durdurmuyor. Durduran şey, sahiplenmenin kaçıncı kez
yapıldığı bilgisi. Eski worker elindeki sayıyla geliyor, güncel sayı ondan
büyük, bildirim reddediliyor.

Fence yenilemede artmıyor, yalnızca sahiplenmede artıyor. Yenilemede artsaydı,
worker'ın elindeki değer kendi yenilemesiyle geçersizleşirdi.

## Sonuçları

- Her sonuç bildirimi bir fence doğrulaması yapıyor, yani ek bir okuma.
  `SELECT ... FOR UPDATE` ile yapılıyor, böylece kontrol ile yazma arasında
  devralma olamıyor.
- Uzun süren işler lease yenilemek zorunda. Yenilemezse, henüz çalışırken
  işi elinden alınır. Yenileme aralığı lease süresinin üçte birinden küçük
  tutuluyor ki bir yenileme kaçırılsa bile ikinci deneme yapılabilsin.

## Bu kararın garanti ETMEDİĞİ şey

Fencing, yönetim tablosundaki metadata güncellemesini korur. Eski worker'ın
harici bir sistemde yan etki üretmesini engellemez.

Somut örnek: eski worker lease'ini kaybetti ama hâlâ çalışıyor ve tenant
şemasında `CREATE SCHEMA` çalıştırıyor. Fencing bunu durdurmaz. Fencing
yalnızca "tamamlandı" yazmasını engeller.

Bunun için ayrıca şunlar gerekir ve henüz yok:

- Kurulum adımlarının güvenle tekrarlanabilir olması (`IF NOT EXISTS` gibi).
- Tenant bazlı bir kilit, ya da hedef sistemin kendi fencing desteği.

Bu yüzden sistem "en fazla bir kez çalışır" garantisi vermiyor. İş yeniden
çalışabilir ve handler'ların bunu kaldırması gerekiyor.

## Bu karar ne zaman yanlış hale gelir

- İş süresi lease süresinden düzenli olarak uzun sürerse. Yenileme bunu
  kapatır ama yenileme de başarısız olabilir; o noktada lease süresini
  uzatmak veya işi parçalara bölmek gerekir.
- Saat kayması ciddi boyuta ulaşırsa. Lease bitişi veritabanı saatiyle
  yazılıyor, bu yüzden şu an worker saatlerine bağımlı değil. Lease kontrolü
  uygulama tarafına taşınırsa bu bağımlılık geri gelir.
