# 🌐 Guide d'Installation

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

> [!TIP]
> **Support Docker/Podman** : Vous pouvez également utiliser Docker ou Podman pour exécuter **whatsabladerunner**. Consultez [DOCKER.md](../DOCKER.md) pour les instructions.

Bienvenue sur **whatsabladerunner** ! 🚀 Ce guide vous aidera à compiler et installer le projet sur votre système. Une fois installé, **whatsabladerunner** supporte 15 langues ! 🌍

## 📋 Prérequis

Avant de compiler le projet, assurez-vous d'avoir installé les éléments suivants :

1. **Go (Golang)** 🐹 : Vous avez besoin de la version 1.21 de Go ou supérieure.
   * **Téléchargement** : [https://go.dev/dl/](https://go.dev/dl/)
   * **Vérification** : Exécutez `go version` dans votre terminal.

2. **Git** 🪵 : Pour récupérer le code source.
   * **Téléchargement** : [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Vérification** : Exécutez `git --version` dans votre terminal.

## 📥 Récupération du Code Source

Ouvrez votre terminal ou invite de commande et exécutez :

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Instructions de Compilation

### 🐧 Linux et 🍎 macOS

1. **Compilation** 🔨 :

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Exécution** ▶️ :

   ```bash
   ./whatsabladerunner
   ```

   *Note : S'il y a un problème de permissions, vous devrez peut-être le rendre exécutable avec `chmod +x whatsabladerunner`.* 🔑

### 🪟 Windows

1. **Compilation** 🔨 :

   Ouvrez l'Invite de Commande ou PowerShell et exécutez :

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Exécution** ▶️ :

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Premier Lancement : Liaison WhatsApp

La première fois que vous lancez **whatsabladerunner**, vous devez le lier à votre compte WhatsApp :

1. **Scanner le Code QR** 🔍 : Un code QR apparaîtra dans votre terminal.
2. **Lier l'Appareil** 🔗 : Sur votre téléphone, ouvrez WhatsApp > Réglages > Appareils liés > Lier un appareil.
3. **Scanner** 📸 : Scannez le code QR du terminal avec votre téléphone.
4. **Infos de l'Appareil** ℹ️ : Une fois lié, il apparaîtra dans vos réglages WhatsApp comme **Google Chrome** sur **Windows**. C'est un comportement normal pour la bibliothèque utilisée. ✅
