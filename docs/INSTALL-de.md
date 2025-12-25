# 🌐 Installationsanleitung

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

> [!TIP]
> **Docker/Podman-Unterstützung**: Du kannst auch Docker oder Gebinde verwenden, um **whatsabladerunner** auszuführen. Siehe [DOCKER.md](../DOCKER.md) für Anweisungen.

Willkommen bei **whatsabladerunner**! 🚀 Diese Anleitung hilft dir beim Kompilieren und Installieren des Projekts auf deinem System. Einmal installiert, unterstützt **whatsabladerunner** 15 Sprachen! 🌍

## 📋 Voraussetzungen

Bevor du das Projekt kompilierst, stelle sicher, dass Folgendes installiert ist:

1. **Go (Golang)** 🐹: Du benötigst Go-Version 1.21 oder höher.
   * **Download**: [https://go.dev/dl/](https://go.dev/dl/)
   * **Verifizierung**: Führe `go version` in deinem Terminal aus.

2. **Git** 🪵: Um den Quellcode zu beziehen.
   * **Download**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Verifizierung**: Führe `git --version` in deinem Terminal aus.

## 📥 Quellcode beziehen

Öffne dein Terminal oder deine Eingabeaufforderung und führe aus:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Installationsanweisungen

### 🐧 Linux und 🍎 macOS

1. **Kompilieren** 🔨:

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Ausführen** ▶️:

   ```bash
   ./whatsabladerunner
   ```

   *Hinweis: Wenn du Berechtigungsprobleme hast, musst du die Datei möglicherweise mit `chmod +x whatsabladerunner` ausführbar machen.* 🔑

### 🪟 Windows

1. **Kompilieren** 🔨:

   Öffne die Eingabeaufforderung oder PowerShell und führe aus:

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Ausführen** ▶️:

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Erster Start: WhatsApp verknüpfen

Wenn du **whatsabladerunner** zum ersten Mal ausführst, musst du es mit deinem WhatsApp-Konto verknüpfen:

1. **QR-Code scannen** 🔍: Ein QR-Code erscheint in deinem Terminal.
2. **Gerät verknüpfen** 🔗: Öffne auf deinem Handy WhatsApp > Einstellungen > Verknüpfte Geräte > Gerät hinzufügen.
3. **Scannen** 📸: Scanne den QR-Code im Terminal mit deinem Handy.
4. **Geräteinfo** ℹ️: Nach der Verknüpfung wird es in deinen WhatsApp-Einstellungen als **Google Chrome** unter **Windows** angezeigt. Dies ist das normale Verhalten der verwendeten Bibliothek. ✅
