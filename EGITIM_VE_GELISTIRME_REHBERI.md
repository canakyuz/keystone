# NexSpaces API Eğitim ve Geliştirme Rehberi

Bu rehber, `nexpaces-api` projesinin mimarisini anlamak, geliştirme ortamını kurup yönetmek ve API dokümantasyonunu sıfırdan oluşturmak için hazırlanmış kapsamlı bir eğitim dokümanıdır.

---

## Bölüm 1: Kavramlar ve Kurulum

Bu bölümde, projenin temelini oluşturan mimari ve teknolojik kavramları anlayacak ve geliştirme ortamını adım adım ayağa kaldıracağız.

### 1.1. Proje Mimarisi: Clean Architecture

Proje, sürdürülebilir ve test edilebilir bir yapı kurmayı amaçlayan **Clean Architecture (Temiz Mimari)** prensibini benimser. Kodlar, sorumluluklarına göre katmanlara ayrılmıştır.

- **`cmd/server/main.go`**: Uygulamanın ana giriş noktasıdır. Sadece sunucuyu başlatmaktan sorumludur.
- **`internal/`**: Projenin ana iş mantığının bulunduğu, dış dünyadan soyutlanmış bölümüdür.
    - **`core/`**: Mimarinin kalbidir. Projenin temel iş kurallarını, varlıklarını (`domain`) ve arayüzlerini (`ports`) içerir. Bu katman, veritabanı veya web framework gibi dış detaylardan tamamen bağımsızdır.
    - **`adapters/`**: `core` katmanındaki arayüzlerin somut olarak hayata geçirildiği yerdir. Örneğin, veritabanı sorgularının yazıldığı (`persistence/`) veya gelen HTTP isteklerinin karşılandığı (`http/`) kodlar burada bulunur.
    - **`app/`**: Uygulamanın farklı katmanlarını bir araya getiren, "yapıştırıcı" görevi gören bölümdür.

### 1.2. Konteyner Teknolojisi: Docker

- **Docker Nedir?** Docker, uygulamaları ve onların tüm bağımlılıklarını "konteyner" adı verilen izole paketlere koyan bir araçtır. Bu sayede "benim makinemde çalışıyordu" sorunu ortadan kalkar.
- **Neden Kullanıyoruz?** Projemiz çalışmak için bir PostgreSQL veritabanına ve bir Redis sunucusuna ihtiyaç duyar. Docker, bu servisleri kendi bilgisayarınıza manuel olarak kurmak yerine, tek bir komutla, proje için özel olarak ayarlanmış versiyonlarıyla birlikte başlatmanızı sağlar.
- **`docker-compose.yml`**: Hangi konteynerlerin (API, veritabanı, Redis) nasıl ayarlarla (`.env` dosyasındaki şifreler gibi) başlatılacağını tanımlayan bir "orkestrasyon" dosyasıdır.

### 1.3. Adım Adım Geliştirme Ortamını Başlatma

**1. Ön Gereksinim: Docker Desktop**
   - Bilgisayarınızda [Docker Desktop](https://www.docker.com/products/docker-desktop/) uygulamasının kurulu ve **çalışıyor** olduğundan emin olun.

**2. Ortam Değişkenleri (`.env`)**
   - Proje, veritabanı şifresi gibi hassas veya ortama göre değişen bilgileri `.env` dosyasından okur. Proje ana dizinindeki `.env.example` dosyasını kopyalayıp adını `.env` olarak değiştirerek kendi ortam dosyanızı oluşturun. Geliştirme için içindeki ayarlar genellikle yeterlidir.

**3. Ortamı Başlatma (`make dev`)**
   - `make`, tekrar eden komutları basitleştiren bir otomasyon aracıdır. `make dev` komutu, arka planda `docker-compose`'u çalıştırarak tüm konteynerleri başlatır.
   ```bash
   # nexpaces-api ana dizinindeyken çalıştırın
   make dev
   ```

**4. Doğrulama ve Takip**
   - **Logları İzleme:** `make logs` komutu ile tüm servislerin loglarını canlı olarak izleyebilir ve bir hata olup olmadığını görebilirsiniz.
   - **Sağlık Kontrolü:** `make health` komutu, API'nin ayakta ve çalışır durumda olup olmadığını kontrol eder.

**5. Ortamı Durdurma**
   - Çalışmayı bitirdiğinizde, `make stop` komutu ile tüm konteynerleri güvenli bir şekilde kapatabilirsiniz.

---

## Bölüm 2: Sıfırdan API Dokümantasyonu Oluşturma (Eğitimi)

Bu bölümde, `swaggo/swag` aracını kullanarak, kodun içindeki yorumlardan nasıl otomatik olarak interaktif bir API dokümantasyonu oluşturulduğunu öğreneceğiz.

### 2.1. Teori: Neden ve Nasıl?

- **API Dokümantasyonu Neden Önemli?** Bir API'nin nasıl kullanılacağını (hangi endpoint'ler var, ne parametre alıyor, ne yanıt dönüyor) anlatan bir kullanım kılavuzudur. Hem frontend geliştiricileri hem de gelecekteki siz için hayat kurtarıcıdır.
- **OpenAPI ve Swagger UI:** OpenAPI, API'leri tanımlamak için kullanılan bir standarttır. Swagger UI ise bu standarda uygun yazılmış bir tanım dosyasını (örn: `swagger.json`) alıp şık ve interaktif bir web sayfasına dönüştüren bir araçtır.
- **`swaggo/swag` Nasıl Çalışır?** Bu araç, projenizdeki Go dosyalarını tarar. Belirli bir formatla (`// @Summary` gibi) yazılmış yorumları bulur, bunları OpenAPI standardına uygun bir `swagger.json` dosyasına çevirir ve bir `docs` paketi oluşturur.

### 2.2. Uygulamalı Örnek: Yeni Bir Endpoint'i Dokümante Etme

Şimdi, var olmayan bir endpoint'i (örneğin, tek bir kullanıcıyı ID ile getiren) hem kodlayıp hem de dokümante edelim.

**1. Handler Fonksiyonunu Yazma (`internal/app/app.go` içinde)**
   - `setupRoutes` fonksiyonuna aşağıdaki gibi yeni bir endpoint eklediğimizi varsayalım:
   ```go
   // users grubunun içine ekleyelim
   users.Get("/:id", func(c *fiber.Ctx) error {
       id := c.Params("id")
       // Normalde burada veritabanından kullanıcıyı buluruz.
       // Şimdilik sadece örnek bir yanıt dönelim.
       return c.JSON(fiber.Map{
           "id": id,
           "name": "John Doe",
           "email": "john.doe@example.com",
       })
   })
   ```

**2. Detaylı Dokümantasyon Yorumları Ekleme**
   - Şimdi bu fonksiyonun hemen üzerine, `swag`'in anlayacağı dilde yorumlar ekleyelim:
   ```go
   // @Summary      Get a single user by ID
   // @Description  Retrieves user details based on their unique identifier.
   // @Tags         users
   // @Accept       json
   // @Produce      json
   // @Param        id   path      string  true  "User ID"
   // @Success      200  {object}  map[string]interface{}
   // @Failure      404  {object}  map[string]string
   // @Router       /api/v1/users/{id} [get]
   users.Get("/:id", ...) // Fonksiyonun devamı
   ```
   - **Her Satırın Anlamı:**
     - `@Summary`: Dokümantasyon arayüzünde görünecek kısa başlık.
     - `@Description`: Endpoint'in ne yaptığına dair daha uzun açıklama.
     - `@Tags`: Bu endpoint'i "users" grubu altında topla.
     - `@Param`: Bir parametre tanımlar.
       - `id`: Parametrenin adı.
       - `path`: Parametrenin yolda (`/users/{id}` gibi) olduğunu belirtir. Diğer seçenekler: `query`, `header`.
       - `string`: Veri tipi.
       - `true`: Bu parametrenin zorunlu olduğunu belirtir.
       - `"User ID"`: Arayüzde görünecek açıklama.
     - `@Success`: Başarılı yanıtı tanımlar. `200` HTTP kodu ile dönecek ve `object` tipinde bir `map[string]interface{}` içerecek.
     - `@Failure`: Hatalı yanıtı tanımlar. `404` (Not Found) durumunda dönecek yanıt.
     - `@Router`: Endpoint'in tam yolunu ve metodunu belirtir. Bu, `swag` için en önemli direktiflerden biridir.

**3. Dokümantasyon Dosyalarını Oluşturma**
   - Yorumları ekledikten sonra, terminalde `swag init` komutunu çalıştırarak `docs` klasörünü ve içindeki dosyaları (yeniden) oluşturun.
   ```bash
   # swag komutunun tam yolunu kullanmayı unutmayın!
   /Users/canakyuz/go/bin/swag init -g cmd/server/main.go
   ```

**4. Değişiklikleri Uygulama ve Doğrulama**
   - Hem yeni kodu hem de yeni dokümantasyon dosyalarını aktif hale getirmek için sunucuyu yeniden başlatın:
   ```bash
   make dev
   ```
   - Tarayıcınızda **http://localhost:8080/swagger/index.html** adresini açın. "users" etiketi altında yeni `/api/v1/users/{id}` endpoint'inizin tüm detaylarıyla eklendiğini göreceksiniz.

---

## Bölüm 3: Sorun Giderme Rehberi

| Hata Mesajı | Ne Anlama Geliyor? | Neden Olur? | Nasıl Çözülür? |
| :--- | :--- | :--- | :--- |
| `Cannot connect to the Docker daemon` | Docker motoruna bağlanılamıyor. | Docker Desktop uygulaması kapalı veya henüz tam olarak başlamamış. | Docker Desktop'ı başlatın ve çalışır duruma gelmesini bekleyin. |
| `bash: swag: command not found` | `swag` komutu sistem tarafından bulunamıyor. | `go install` ile yüklenen komutların bulunduğu dizin (`$GOPATH/bin`), sistemin arama yaptığı yollar (`PATH`) listesinde değil. | Komutu, `go env GOPATH` ile öğrendiğiniz tam yoluyla çalıştırın: `/Users/canakyuz/go/bin/swag ...` |
| Build sırasında `requires go >= 1.25` gibi bir hata | Bir bağımlılık, Docker imajındaki Go versiyonundan daha yenisini istiyor. | Proje bağımlılıkları zamanla güncellenir ve daha yeni dil özellikleri gerektirebilir. | `infra/docker/Dockerfile.dev` dosyasındaki `FROM golang:1.24-alpine` satırını, istenen versiyonla (`FROM golang:1.25-alpine`) güncelleyin. |
| `cannot parse source files ... main.go: no such file or directory` | `swag init` komutu, `main.go` dosyasını bulamadı. | `swag`, varsayılan olarak `main.go`'yu ana dizinde arar, ancak bizim projemizde `cmd/server/` altındadır. | `swag init` komutunu `-g cmd/server/main.go` parametresi ile çalıştırarak `main.go`'nun doğru yerini belirtin. |

