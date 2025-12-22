# Installation Guide

Welcome to **WhatsABladeRunner**! This guide will help you build and install the project on your system.

## Prerequisites

Before building the project, ensure you have the following installed:

1.  **Go (Golang)**: You need Go version 1.21 or higher.
    *   **Download**: [https://go.dev/dl/](https://go.dev/dl/)
    *   **Verify**: Run `go version` in your terminal.

2.  **Git**: To fetch the source code.
    *   **Download**: [https://git-scm.com/downloads](https://git-scm.com/downloads)
    *   **Verify**: Run `git --version` in your terminal.

## Getting the Source Code

Open your terminal or command prompt and run:

```bash
git clone https://github.com/minimasoft/whatsabladerunner.git
cd whatsabladerunner
```

## Build Instructions

### Linux and macOS

1.  **Build**:
    ```bash
    go build -o whatsabladerunner main.go
    ```
2.  **Run**:
    ```bash
    ./whatsabladerunner
    ```

    *Note: If you encounter permission issues, you might need to make it executable with `chmod +x whatsabladerunner`.*

### Windows

1.  **Build**:
    Open Command Prompt or PowerShell and run:
    ```powershell
    go build -o whatsabladerunner.exe main.go
    ```
2.  **Run**:
    ```powershell
    .\whatsabladerunner.exe
    ```

---
*Happy Vibes!*
