# 🌐 Guía de Instalación

🌐 [English](INSTALL.md) | 🇦🇷 [Español](INSTALL-es.md) | 🇧🇷 [Português](INSTALL-pt.md) | 🇫🇷 [Français](INSTALL-fr.md) | 🇮🇹 [Italiano](INSTALL-it.md) | 🇩🇪 [Deutsch](INSTALL-de.md) | 🇮🇳 [हिन्दी](INSTALL-hi.md) | 🇮🇩 [Bahasa Indonesia](INSTALL-id.md)

¡Bienvenido a **whatsabladerunner**! 🚀 Esta guía te ayudará a compilar e instalar el proyecto en tu sistema. ¡Una vez instalado, **whatsabladerunner** soporta 15 idiomas! 🌍

## 📋 Prerrequisitos

Antes de compilar el proyecto, asegúrate de tener instalado lo siguiente:

1. **Go (Golang)** 🐹: Necesitas la versión 1.21 o superior de Go.
   * **Descarga**: [https://go.dev/dl/](https://go.dev/dl/)
   * **Verificación**: Ejecuta `go version` en tu terminal.

2. **Git** 🪵: Para obtener el código fuente.
   * **Descarga**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
   * **Verificación**: Ejecuta `git --version` en tu terminal.

## 📥 Obtener el Código Fuente

Abre tu terminal o símbolo del sistema y ejecuta:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## 🛠️ Instrucciones de Compilación

### 🐧 Linux y 🍎 macOS

1. **Compilar** 🔨:

   ```bash
   go build -o whatsabladerunner main.go
   ```

2. **Ejecutar** ▶️:

   ```bash
   ./whatsabladerunner
   ```

   *Nota: Si encuentras problemas de permisos, es posible que necesites hacerlo ejecutable con `chmod +x whatsabladerunner`.* 🔑

### 🪟 Windows

1. **Compilar** 🔨:

   Abre el Símbolo del sistema o PowerShell y ejecuta:

   ```powershell
   go build -o whatsabladerunner.exe main.go
   ```

2. **Ejecutar** ▶️:

   ```powershell
   .\whatsabladerunner.exe
   ```

## 📱 Primera Ejecución: Vincular WhatsApp

La primera vez que ejecutas **whatsabladerunner**, debes vincularlo a tu cuenta de WhatsApp:

1. **Escanear Código QR** 🔍: Un código QR aparecerá en tu terminal.
2. **Vincular Dispositivo** 🔗: En tu teléfono, abre WhatsApp > Ajustes > Dispositivos vinculados > Vincular un dispositivo.
3. **Escanear** 📸: Escanea el código QR de la terminal con tu teléfono.
4. **Información del Dispositivo** ℹ️: Una vez vinculado, aparecerá en tus ajustes de WhatsApp como **Google Chrome** en **Windows**. Este es el comportamiento normal de la librería utilizada. ✅
