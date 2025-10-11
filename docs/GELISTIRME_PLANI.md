# NexSpaces - Geliştirme Planı

**Güncelleme:** 11 Ekim 2025

---

## 🎯 Öncelik Sırası

```
1. CMS/Website        → 3-4 hafta    (ŞİMDİ)
2. LMS                → 3-4 hafta
3. CRM                → 4-5 hafta
4. E-ticaret          → 5-6 hafta
5. ERP                → 10-12 hafta
6. OMS                → 8-10 hafta
7. HMS                → 8-10 hafta

Toplam: ~50 hafta (12 ay)
MVP (1-4): 19 hafta (5 ay)
```

---

## 🧩 Araçlar (Tools)

Modüller arasında paylaşılan özellikler:

### İçerik Araçları
- **Zengin Metin Editörü** - Blog, ders notları, ürün açıklaması
- **Medya Kütüphanesi** - Resim/video yükleme, S3 entegrasyonu
- **SEO Yöneticisi** - Meta tag, sitemap, robots.txt

### İletişim Araçları
- **E-posta Servisi** - Hoşgeldin, hatırlatma, sipariş maili
- **Bildirim Merkezi** - Uygulama içi bildirimler, hatırlatmalar
- **Chat/Mesajlaşma** - Canlı destek, öğrenci-öğretmen iletişim

### Ticaret Araçları
- **Ödeme Sistemi** - Stripe, iyzico, Checkout.com (✅ VAR)
- **İade Yönetimi** - Tam/kısmi iade (✅ VAR)
- **Abonelik Yönetimi** - Aylık/yıllık faturalandırma
- **Fatura Oluşturucu** - PDF fatura, vergi hesaplama

### Yönetim Araçları
- **Takvim/Randevu** - Ders, toplantı, rezervasyon (✅ VAR)
- **Görev Yöneticisi** - To-do, takip, hatırlatma
- **Form Oluşturucu** - İletişim formu, kayıt formu

### Entegrasyon Araçları
- **Webhook Yöneticisi** - Olay bildirimleri (✅ VAR)
- **N8n Otomasyonu** - İş akışı otomasyonu
- **API Gateway** - Rate limiting, log (✅ VAR)

### Analitik Araçları
- **Dashboard** - Satış, kullanıcı, gelir metrikleri
- **Raporlama** - Excel/PDF export, özel raporlar

---

## 📦 Modüller

### 1. CMS/Website (3-4 hafta) ← **ŞİMDİ**

**Ne yapar:** Blog, web sitesi, landing page oluşturma

**Özellikler:**
- Sayfa oluşturma (taslak, yayında, arşiv)
- Blog yazıları, kategoriler
- Şablon sistemi (marketplace'den seç)
- Özel alan adı (customer.com)
- SEO optimizasyonu

**Kullandığı Araçlar:**
- Zengin Metin Editörü ✅
- Medya Kütüphanesi ✅
- SEO Yöneticisi ✅
- E-posta Servisi ✅

**Hedef Müşteri:** Küçük işletmeler, bloggerlar, freelancerlar

---

### 2. LMS (3-4 hafta)

**Ne yapar:** Online ders platformu, özel ders yönetimi

**Özellikler:**
- Ders planlama (birebir, grup, online)
- Ödev takibi, not verme
- Öğrenci performans raporu
- Zoom/Meet entegrasyonu
- Ders hatırlatmaları (24 saat, 1 saat öncesi)
- Takvim export (iCal)

**Kullandığı Araçlar:**
- Takvim/Randevu ✅
- Bildirim Merkezi ✅
- E-posta Servisi ✅
- Zengin Metin Editörü ✅
- Ödeme Sistemi ✅ (ücretli dersler için)

**Hedef Müşteri:** Özel ders hocaları, eğitim kurumları, online kurslar

---

### 3. CRM (4-5 hafta)

**Ne yapar:** Müşteri ilişkileri yönetimi, satış takibi

**Özellikler:**
- Müşteri/Lead yönetimi
- Satış hunisi (pipeline)
- Aktivite takibi (arama, email, toplantı)
- Görev yönetimi (follow-up)
- E-posta entegrasyonu
- Satış raporları

**Kullandığı Araçlar:**
- Görev Yöneticisi ✅
- E-posta Servisi ✅
- Takvim/Randevu ✅
- Dashboard ✅

**Hedef Müşteri:** Satış ekipleri, B2B şirketler, danışmanlar

---

### 4. E-ticaret (5-6 hafta)

**Ne yapar:** Online mağaza, ürün satışı

**Özellikler:**
- Ürün kataloğu (varyant, stok)
- Sepet, ödeme, sipariş
- Kargo entegrasyonu
- İndirim kodları
- Sipariş takibi
- İade/değişim yönetimi

**Kullandığı Araçlar:**
- Ödeme Sistemi ✅
- İade Yönetimi ✅
- Medya Kütüphanesi ✅
- Fatura Oluşturucu ✅
- E-posta Servisi ✅

**Hedef Müşteri:** Online satıcılar, mağazalar, dropshipping

---

### 5. ERP (10-12 hafta)

**Ne yapar:** Kurumsal kaynak planlama - muhasebe, stok, İK, üretim

**Alt Modüller (10 tane):**
1. **Muhasebe** - Cari hesap, kasa, banka, mali raporlar
2. **Stok Yönetimi** - Çok depolu stok, barkod, stok hareketleri
3. **Tedarik Zinciri** - Tedarikçi, satın alma siparişi
4. **Üretim** - Ürün reçetesi, üretim emri, kapasite planlama
5. **İnsan Kaynakları** - Personel, bordro, izin, performans
6. **Varlık Yönetimi** - Sabit kıymet, amortisman, bakım
7. **Satış Yönetimi** - Satış siparişi, teklif, komisyon
8. **Proje Yönetimi** - Gantt, kaynak planlama, zaman takibi
9. **Doküman Yönetimi** - Dosya saklama, versiyon, imza
10. **Raporlama** - Özel rapor oluşturucu, dashboard

**Yeni Araçlar (8 tane):**
- Muhasebe Motoru (çift taraflı kayıt)
- Bordro Motoru (maaş, kesinti)
- Üretim Planlayıcı (Gantt)
- Barkod Okuyucu Entegrasyonu
- Banka Senkronizasyonu
- Dijital İmza (DocuSign)
- Amortisman Hesaplayıcı
- MRP Hesaplayıcı (malzeme ihtiyaç)

**Hedef Müşteri:** Fabrikalar, üretim şirketleri, holding şirketleri

---

### 6. OMS (8-10 hafta)

**Ne yapar:** Sipariş yönetim sistemi - çok kanallı sipariş, kargolama

**Alt Modüller (5 tane):**
1. **Sipariş İşleme** - Çok kanallı sipariş (web, mobil, telefon)
2. **Kargolama** - Toplama, paketleme, etiket, kargo takibi
3. **Stok Tahsisi** - Akıllı depo seçimi, rezervasyon
4. **İade/Değişim** - RMA numarası, iade süreci
5. **B2B Portal** - Toptan müşteri, özel fiyat, toplu sipariş

**Yeni Araçlar (6 tane):**
- Sipariş Yönlendirme (akıllı depo seçimi)
- ATP Hesaplayıcı (stok uygunluk)
- Kargo Etiketi Oluşturucu (UPS/DHL)
- Toplama/Paketleme İş Akışı
- RMA İş Akışı
- B2B Portal

**Hedef Müşteri:** E-ticaret şirketleri, distribütörler, toptancılar

---

### 7. HMS (8-10 hafta)

**Ne yapar:** Otel yönetim sistemi - rezervasyon, resepsiyon, kat hizmetleri

**Alt Modüller (7 tane):**
1. **Mülk Yönetimi** - Otel, oda tipleri, oda durumu
2. **Rezervasyon** - Online rezervasyon, fiyat planları
3. **Resepsiyon** - Check-in/out, oda ataması, misafir hesabı
4. **Kat Hizmetleri** - Temizlik programı, görev ataması
5. **Misafir Hizmetleri** - Misafir profili, sadakat programı
6. **Restoran/Kasa** - Masa yönetimi, sipariş, odaya işleme
7. **Kanal Yönetimi** - Booking.com, Airbnb entegrasyonu

**Yeni Araçlar (7 tane):**
- Kanal Yöneticisi (OTA entegrasyon)
- Fiyat Yönetimi (dinamik fiyat)
- Misafir Hesap Sistemi (folio)
- POS Sistemi (restoran kasası)
- Kapı Kartı Entegrasyonu (RFID)
- Kat Hizmetleri Mobil Uygulama
- Misafir Portalı

**Hedef Müşteri:** Oteller, apart oteller, butik oteller, tatil köyleri

---

## 📊 Hangi Modül Hangi Aracı Kullanır?

| ARAÇ             | CMS | LMS | CRM | E-ticaret| ERP | OMS | HMS |
|------------------|-----|-----|-----|----------|-----|-----|-----|
| Zengin Editör    |  ✅ |  ✅ |  ✅ |    ✅    |  ✅ |  -  |  -  |
| Medya Kütüphane  |  ✅ |  ✅ |  -  |    ✅    |  -  |  -  |  ✅ |
| SEO Yöneticisi   |  ✅ |  -  |  -  |    ✅    |  -  |  -  |  -  |
| E-posta Servisi  |  ✅ |  ✅ |  ✅ |    ✅    |  ✅ |  ✅ |  ✅ |
| Bildirim Merkezi |  -  |  ✅ |  ✅ |    ✅    |  ✅ |  ✅ |  ✅ |
| Ödeme Sistemi    |  -  |  ✅ |  -  |    ✅    |  ✅ |  ✅ |  ✅ |
| Takvim/Randevu   |  -  |  ✅ |  ✅ |    -     |  ✅ |  -  |  ✅ |
| Görev Yöneticisi |  -  |  -  |  ✅ |    -     |  ✅ |  -  |  ✅ |
| Dashboard        |  ✅ |  ✅ |  ✅ |    ✅    |  ✅ |  ✅ |  ✅ |


---

## 💰 Fiyatlandırma (Modüler)

```
Temel:        CMS + Blog                    → 99 TL/ay
Profesyonel:  + LMS + Randevu               → 249 TL/ay
İş:           + CRM + E-ticaret             → 499 TL/ay
Kurumsal:     + ERP + OMS + HMS             → 1.499 TL/ay
Özel:         İstediğin modülleri seç       → Özel fiyat
```

---

## 🚀 Şimdi Ne Yapacağız?

### **Bu Hafta: CMS - Week 1 (4 gün)**

**Gün 1-2: Website Entity Testleri**
- Dosya: `internal/domain/website/entity_test.go` (YENİ)
- Test kapsamı: %0 → %80+
- 12+ test senaryosu

**Gün 3-4: Website Repository**
- Dosya: `internal/repository/postgres/website_repository.go` (YENİ)
- PostgreSQL + RLS
- CRUD işlemleri
- Test kapsamı: %80+

**Gün 5: Website API**
- Dosya: `internal/handler/website/handler.go`
- REST API endpoints (7 tane)
- Swagger dokümantasyonu

---

## 📈 İstatistikler

```
Toplam Modül:      9 (CMS, LMS, CRM, E-ticaret, ERP, OMS, HMS, Randevu, Portfolio)
Alt Modül:         38
Toplam Araç:       65+
Geliştirme Süresi: 50 hafta (12 ay)
MVP Süresi:        19 hafta (5 ay) - İlk 4 modül
```

---

## ✅ Mevcut Durum

```
✅ Tamamlandı:
  - User domain (%89 test)
  - Tenant domain (%53 test)
  - Payment domain (entity hazır)
  - Refund domain (entity hazır)
  - Webhook domain (entity hazır)
  - Booking domain (entity hazır)

🚧 Şimdi:
  - CMS/Website (Week 1 başlıyor)

⏳ Sırada:
  - Blog domain testleri
  - LMS domain testleri
  - CRM domain (yeni)
  - E-ticaret domain (yeni)
```

---

**Hazır mıyız? CMS geliştirmeye başlayalım!** 🎯
