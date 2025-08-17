# Go HTTP Server

Yengil va tezkor HTTP server --- Go tilida yozilgan.\
Statik fayllar serving, request logging, server statistikasi va graceful
shutdown kabi funksiyalarni qo'llab-quvvatlaydi.

------------------------------------------------------------------------

## ✨ Xususiyatlar

-   🚀 **Asosiy funksiyalar**
    -   HTTP/1.1 protokoli qo'llab-quvvatlash\
    -   Bir vaqtda bir nechta connectionlarni boshqarish (goroutines)\
    -   Statik fayllarni tez va xavfsiz serve qilish\
    -   MIME type avtomatik aniqlash\
    -   Directory traversal hujumlariga qarshi himoya\
    -   Request logging va server statistikasi\
    -   Graceful shutdown (toza yopilish)
-   📊 **Performance**
    -   Goroutines sababli yuqori tezlik\
    -   Low memory footprint\
    -   Efficient concurrency modeli\
    -   Optimal fayl serve qilish

------------------------------------------------------------------------

## 🚀 Tez boshlash

### 1. Sample website yaratish

``` bash
go run main.go --setup
```

### 2. Serverni ishga tushirish

``` bash
go run main.go
```

### 3. Binary build qilish

``` bash
make build
./httpd
```

------------------------------------------------------------------------

## ⚙️ Command Line Options

``` bash
# Portni o‘zgartirish
go run main.go -p 3000

# Custom document root
go run main.go -r /var/www

# Yordam
go run main.go --help
```

------------------------------------------------------------------------

## 🧪 Test qilish

``` bash
# Browser orqali
http://localhost:8080

# curl orqali
curl http://localhost:8080
curl http://localhost:8080/api.json
curl http://localhost:8080/test.html
```

------------------------------------------------------------------------

## 🛠️ Makefile Commands

``` bash
make setup         # Sample website yaratish
make run           # Server ishga tushirish
make build         # Binary build qilish
make build-all     # Cross-platform build
make docker-build  # Docker image build
```

------------------------------------------------------------------------

## 📂 Loyihaning tuzilishi

    .
    ├── main.go          # Asosiy HTTP server kodi
    ├── Makefile         # Build va run uchun buyruqlar
    ├── www/             # Statik fayllar (document root)
    └── README.md        # Hujjat

------------------------------------------------------------------------

⚡ Made with Go ❤️