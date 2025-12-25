# 🌐 Panduan Instalasi

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

Selamat datang di **whatsabladerunner**! 🚀 Panduan ini akan membantu Anda membangun dan menginstal proyek di sistem Anda. Setelah terinstal, **whatsabladerunner** mendukung 15 bahasa! 🌍

## 📋 Prasyarat

Sebelum membangun proyek, pastikan Anda telah menginstal hal-hal berikut:

1. **Go (Golang)** 🐹: Anda memerlukan Go versi 1.21 atau lebih tinggi.
   * **Unduh**: [https://go.dev/dl/](https://go.dev/dl/)
   * **Verifikasi**: Jalankan `go version` di terminal Anda.

2. **Git** 🪵: Untuk mengambil kode sumber.
   * **Unduh**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Verifikasi**: Jalankan `git --version` di terminal Anda.

## 📥 Mendapatkan Kode Sumber

Buka terminal atau command prompt Anda dan jalankan:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Instruksi Pembuatan

### 🐧 Linux dan 🍎 macOS

1. **Build** 🔨:

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Jalankan** ▶️:

   ```bash
   ./whatsabladerunner
   ```

   *Catatan: Jika Anda mengalami masalah izin, Anda mungkin perlu membuatnya dapat dieksekusi dengan `chmod +x whatsabladerunner`.* 🔑

### 🪟 Windows

1. **Build** 🔨:

   Buka Command Prompt atau PowerShell dan jalankan:

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Jalankan** ▶️:

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Jalankan Pertama Kali: Menghubungkan WhatsApp

Pertama kali Anda menjalankan **whatsabladerunner**, Anda harus menghubungkannya ke akun WhatsApp Anda:

1. **Pindai Kode QR** 🔍: Kode QR akan muncul di terminal Anda.
2. **Tautkan Perangkat** 🔗: Di ponsel Anda, buka WhatsApp > Pengaturan > Perangkat Tautkan > Tautkan Perangkat.
3. **Pindai** 📸: Pindai kode QR terminal dengan ponsel Anda.
4. **Info Perangkat** ℹ️: Setelah terhubung, itu akan muncul di pengaturan WhatsApp Anda sebagai **Google Chrome** di **Windows**. Ini adalah perilaku normal bagi pustaka yang digunakan. ✅
