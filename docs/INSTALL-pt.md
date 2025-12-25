# 🌐 Guia de Instalação

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

Bem-vindo ao **whatsabladerunner**! 🚀 Este guia ajudará você a compilar e instalar o projeto em seu sistema. Depois de instalado, o **whatsabladerunner** suporta 15 idiomas! 🌍

## 📋 Pré-requisitos

Antes de compilar o projeto, certifique-se de ter o seguinte instalado:

1. **Go (Golang)** 🐹: Você precisa da versão 1.21 ou superior do Go.
   * **Download**: [https://go.dev/dl/](https://go.dev/dl/)
   * **Verificar**: Execute `go version` no seu terminal.

2. **Git** 🪵: Para baixar o código-fonte.
   * **Download**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Verificar**: Execute `git --version` no seu terminal.

## 📥 Obtendo o Código-Fonte

Abra seu terminal ou prompt de comando e execute:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Instruções de Compilação

### 🐧 Linux e 🍎 macOS

1. **Compilar** 🔨:

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Executar** ▶️:

   ```bash
   ./whatsabladerunner
   ```

   *Nota: Se você encontrar problemas de permissão, pode ser necessário torná-lo executável com `chmod +x whatsabladerunner`.* 🔑

### 🪟 Windows

1. **Compilar** 🔨:

   Abra o Prompt de Comando ou PowerShell e execute:

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Executar** ▶️:

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Primeira Execução: Vinculando o WhatsApp

A primeira vez que você executar o **whatsabladerunner**, deverá vinculá-lo à sua conta do WhatsApp:

1. **Escanear QR Code** 🔍: Um QR code aparecerá no seu terminal.
2. **Vincular Dispositivo** 🔗: No seu celular, abra o WhatsApp > Configurações > Aparelhos conectados > Conectar um aparelho.
3. **Escanear** 📸: Escaneie o QR code do terminal com o seu celular.
4. **Info do Aparelho** ℹ️: Depois de conectado, ele aparecerá nas configurações do seu WhatsApp como **Google Chrome** no **Windows**. Este é o comportamento normal da biblioteca utilizada. ✅
