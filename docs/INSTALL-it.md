# 🌐 Guida all'Installazione

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

> [!TIP]
> **Supporto Docker/Podman**: Puoi anche usare Docker o Podman per eseguire **whatsabladerunner**. Consulta [DOCKER.md](../DOCKER.md) per le istruzioni.

Benvenuto in **whatsabladerunner**! 🚀 Questa guida ti aiuterà a compilare e installare il progetto sul tuo sistema. Una volta installato, **whatsabladerunner** supporta 15 lingue! 🌍

## 📋 Prerequisiti

Prima di compilare il progetto, assicurati di avere installato quanto segue:

1. **Go (Golang)** 🐹: È necessaria la versione 1.21 di Go o superiore.
   * **Download**: [https://go.dev/dl/](https://go.dev/dl/)
   * **Verifica**: Esegui `go version` nel tuo terminale.

2. **Git** 🪵: Per scaricare il codice sorgente.
   * **Download**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Verifica**: Esegui `git --version` nel tuo terminale.

## 📥 Scaricare il Codice Sorgente

Apri il tuo terminale o prompt dei comandi ed esegui:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Istruzioni di Compilazione

### 🐧 Linux e 🍎 macOS

1. **Compila** 🔨:

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Esegui** ▶️:

   ```bash
   ./whatsabladerunner
   ```

   *Nota: se riscontri problemi di permessi, potresti doverlo rendere eseguibile con `chmod +x whatsabladerunner`.* 🔑

### 🪟 Windows

1. **Compila** 🔨:

   Apri il Prompt dei Comandi o PowerShell ed esegui:

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Esegui** ▶️:

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Primo Avvio: Collegamento a WhatsApp

La prima volta che esegui **whatsabladerunner**, devi collegarlo al tuo account WhatsApp:

1. **Scansiona il Codice QR** 🔍: Un codice QR apparirà nel tuo terminale.
2. **Collega Dispositivo** 🔗: Sul tuo telefono, apri WhatsApp > Impostazioni > Dispositivi collegati > Collega un dispositivo.
3. **Scansiona** 📸: Scansiona il codice QR del terminale con il tuo telefono.
4. **Info sul Dispositivo** ℹ️: Una volta collegato, apparirà nelle impostazioni di WhatsApp come **Google Chrome** su **Windows**. Questo è il comportamento normale per la libreria utilizzata. ✅
