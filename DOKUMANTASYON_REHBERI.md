# NexSpaces API Geliştirme ve Dokümantasyon Yönetimi Rehberi

Bu doküman, `nexpaces-api` projesini geliştirme ortamında çalıştırmak, test etmek ve API dokümantasyonunu yönetmek için gerekli adımları ve komutları içerir.

## 1. Geliştirme Ortamının Yönetimi (Docker ile)

Proje, tüm bağımlılıkları (PostgreSQL, Redis) ile birlikte bir Docker ortamında çalışacak şekilde tasarlanmıştır. Bu, kurulumu oldukça basitleştirir.

### Ön Gereksinimler
- Makinenizde **Docker Desktop** uygulamasının kurulu ve çalışır durumda olması gerekmektedir.

### Adım 1: Ortam Değişkenlerini Ayarlama
Proje, ayarlarını `.env` dosyasından okur. Projeyi ilk kez kuruyorsanız, ana dizinde bulunan `.env.example` dosyasını kopyalayarak `.env` adında yeni bir dosya oluşturun. Geliştirme ortamı için genellikle içerisindeki varsayılan ayarları değiştirmeniz gerekmez.

### Adım 2: Geliştirme Ortamını Başlatma
Aşağıdaki komut, API sunucusunu, veritabanını ve Redis'i Docker içinde başlatır. Kodda bir değişiklik yaptığınızda sunucunun otomatik olarak yeniden başlamasını sağlayan `air` aracı da bu ortamda aktiftir.

```bash
# nexpaces-api ana dizinindeyken çalıştırın
make dev
```

### Adım 3: Servislerin Loglarını İzleme
Çalışan servislerin (API, veritabanı vb.) loglarını canlı olarak görmek için:

```bash
make logs
```

### Adım 4: Geliştirme Ortamını Durdurma
Tüm servisleri kapatmak için:

```bash
make stop
```

---

## 2. API Dokümantasyonunu Yönetme (Swagger/swag)

Proje, Go kodunun içindeki yorumlardan otomatik olarak interaktif bir web arayüzü (Swagger UI) oluşturan `swaggo/swag` aracını kullanır.

### Adım 1: Endpoint'e Dokümantasyon Yorumları Ekleme
Dokümantasyonda görünmesini istediğiniz bir API endpoint'inin (ilgili handler fonksiyonunun) hemen üzerine aşağıdaki formata uygun yorumlar ekleyin.

**Örnek: `/health` endpoint'i için**
```go
// @Summary Sunucunun durumunu gösterir.
// @Description Sunucunun çalışıp çalışmadığını kontrol eder.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
app.Get("/health", ...)
```
- `@Summary`: İşlemin kısa başlığı.
- `@Description`: İşlemin detaylı açıklaması.
- `@Tags`: Endpoint'leri gruplamak için kullanılan etiket.
- `@Produce`: Döneceği yanıtın formatı (örn: `json`).
- `@Success`: Başarılı durumda dönecek HTTP kodu ve yanıtın formatı.
- `@Router`: Endpoint'in yolu ve HTTP metodu (`[get]`, `[post]` vb.).

### Adım 2: Dokümantasyon Dosyalarını Oluşturma
Yorumları ekledikten sonra, `swag` aracını kullanarak dokümantasyon dosyalarını (`docs` klasörü) oluşturmanız gerekir.

**Önemli:** `swag` komutu sistem `PATH`'inizde bulunmuyorsa, aşağıdaki gibi tam yoluyla çalıştırmanız gerekir.

```bash
# nexpaces-api ana dizinindeyken çalıştırın
/Users/canakyuz/go/bin/swag init -g cmd/server/main.go
```
- `-g cmd/server/main.go`: Dokümantasyonun ana bilgilerini alacağı `main.go` dosyasının yolunu belirtir.

### Adım 3: Değişiklikleri Uygulama
Yeni oluşturulan dokümantasyon dosyaları ve yaptığınız kod değişikliklerinin aktif olması için geliştirme ortamını yeniden başlatmanız gerekir.

```bash
make dev
```

### Adım 4: Dokümantasyonu Görüntüleme
Tarayıcınızdan aşağıdaki adresi ziyaret ederek interaktif API dokümantasyonunu görebilirsiniz:
**http://localhost:8080/swagger/index.html**

---

## 3. Sık Karşılaşılan Sorunlar ve Çözümleri

**1. Hata: `Cannot connect to the Docker daemon...`**
   - **Neden:** Docker Desktop uygulaması çalışmıyor.
   - **Çözüm:** Bilgisayarınızda Docker Desktop'ı başlatın ve tamamen açılmasını bekleyin.

**2. Hata: `bash: swag: command not found`**
   - **Neden:** `swag` komut satırı aracı, sistemin otomatik olarak arama yaptığı klasörlerde (`PATH`) bulunmuyor.
   - **Çözüm:** Komutu, Go'nun kurulum dizinindeki tam yoluyla çalıştırın: `/Users/canakyuz/go/bin/swag init ...`

**3. Hata: Build sırasında Go versiyonu uyumsuzluğu.**
   - **Neden:** Projenin bir bağımlılığı (örn: `air`), projenin Docker imajında belirtilen Go versiyonundan daha yeni bir versiyon gerektiriyor.
   - **Çözüm:** `infra/docker/Dockerfile.dev` dosyasını açın ve `FROM golang:1.24-alpine` gibi olan en üst satırdaki Go versiyonunu, hata mesajında istenen versiyona (örn: `1.25`) yükseltin: `FROM golang:1.25-alpine`.

---

## 4. Yararlı Go Komutları

- **`go get <paket-adı>`**: Projeye yeni bir bağımlılık (kütüphane) ekler.
- **`go install <paket-adı>`**: Bir komut satırı aracı yükler (genellikle `$GOPATH/bin` içine).
- **`go mod tidy`**: `go.mod` dosyasını, koddaki `import` ifadeleriyle senkronize eder, kullanılmayanları temizler.
- **`go env GOPATH`**: Go'nun ana çalışma dizininin (`GOPATH`) yolunu gösterir. Komut satırı araçlarının kurulduğu `bin` klasörü genellikle bu dizinin içindedir.
