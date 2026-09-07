# 0001. Worker veritabanına doğrudan erişir

**Durum:** Kabul edildi, 2026-09-07
**Not:** Bu karar, hedef mimariden bilinçli bir sapmadır.

## Bağlam

Kurulum işleri uzun sürer, farklı veritabanı yetkileri gerektirir ve kullanıcı
isteklerinden bağımsız bir eşzamanlılık sınırına ihtiyaç duyar. Bu üç sebeple
worker'ın Control API'den ayrı bir süreç olması gerekiyor.

Ayrı süreç olması, worker'ın iş kuyruğuna nasıl eriştiği sorusunu açıyor.

Hedef mimari, worker'ın yönetim tablolarına doğrudan erişmemesini, işi
sahiplenmek ve sonuç bildirmek için Control API ile gRPC üzerinden
konuşmasını öneriyor.

## Korunması gereken kurallar

- Bir iş için yalnızca güncel lease sahibi sonuç kaydedebilir.
- Aynı işi iki worker aynı anda yürütemez.
- Süreç iş ortasında kapanırsa iş yeniden devralınabilir olmalıdır.

## Değerlendirilen yaklaşımlar

### A. Worker veritabanına doğrudan erişir

İş sahiplenme, lease yenileme ve sonuç bildirme tek bir SQL ifadesiyle yapılır.

Güçlü tarafı: sahiplenme atomikliği PostgreSQL'in kendi kilit mekanizmasına
düşer. `FOR UPDATE SKIP LOCKED` bunu tek adımda çözer. Control API düşse bile
kuyruk işlemeye devam eder.

Zayıf tarafı: worker yönetim şemasının biçimine bağımlı olur. Şema değişikliği
iki bileşeni birden etkiler. Worker'ın veritabanı yetkisi API'ninkiyle aynı
seviyeye yaklaşır, dolayısıyla ayrı yetki sınırı avantajı zayıflar.

### B. Worker gRPC üzerinden Control API ile konuşur

Sahiplenme, yenileme ve bildirme birer RPC olur.

Güçlü tarafı: yönetim tablolarına erişim tek bir yerde toplanır. Worker'ın
veritabanı yetkisi yalnızca tenant şemalarıyla sınırlanabilir. Sözleşme
açık ve sürümlenebilir olur.

Zayıf tarafı: worker artık API'nin erişilebilirliğine bağımlıdır. API düşerse
kuyruk durur. Ayrıca sahiplenme atomikliği bir ağ çağrısının arkasına geçer;
RPC yanıtı kaybolduğunda worker işi aldı mı almadı mı bilemez ve bu belirsizlik
ayrıca çözülmelidir.

## Karar

Şimdilik A. Worker veritabanına doğrudan erişiyor.

## Gerekçe

Sahiplenme, bu sistemdeki en kritik atomiklik noktası. Bir ağ çağrısının
arkasına koymak, çözülmüş bir problemi yeniden açıyor: RPC yanıtı kaybolursa
worker işi sahiplendi mi bilemez, ve bunu çözmek için gRPC katmanında ikinci
bir idempotency mekanizması gerekir.

B'nin asıl kazancı yetki ayrımı. Ama o kazanç, worker için ayrı bir veritabanı
rolü tanımlanarak da elde edilebilir ve bu daha ucuz. Henüz yapılmadı.

## Sonuçları

- Worker ve Control API aynı yönetim şemasına bağlı. Şema değişikliği ikisini
  birden ilgilendirir.
- Polyrepo'ya ayrılırsa bu iki depo bir şema sürümünü paylaşmak zorunda kalır.
  İş mantığını ortak paketten paylaşmamak yeterli değil, şema sürümü de bir
  bağımlılıktır.
- Yetki ayrımı henüz yok. Worker ve API şu an aynı rolle bağlanıyor.

## Bu karar ne zaman yanlış hale gelir

- Worker üçüncü taraflarca çalıştırılacaksa. O zaman veritabanı erişimi
  verilemez ve B zorunlu olur.
- Yönetim şeması sık değişmeye başlarsa. İki bileşeni birden kırma maliyeti,
  RPC sözleşmesini sürdürme maliyetini geçer.
- Worker sayısı, veritabanı bağlantı bütçesini zorlarsa. API üzerinden
  havuzlamak tek çıkış yolu olur.

## Kapanış koşulu

Worker için ayrı ve dar yetkili bir veritabanı rolü tanımlanana kadar bu karar
eksik uygulanmış sayılır. Yetki ayrımı bu kararın ön koşuluydu.
