# Vigilix 

**Vigilix** is a modern, high-performance terminal user interface (TUI) for managing systemd services. Built with Go and the Bubble Tea framework, it provides a beautiful, responsive, and efficient way to monitor and control your system units.

![Vigilix Demo](https://via.placeholder.com/800x400.png?text=Vigilix+TUI+Screenshot)

## Features

- **Real-Time Monitoring**: View the status of all systemd units instantly.
- **Interactive Control**: Start, stop, and restart services with a single keystroke.
- **Log Streaming**: Watch service logs live as they happen.
- **Config Viewer**: Inspect unit configuration files directly in the terminal.
- **Pro Aesthetics**: Sleek, modern design with custom themes and visual indicators.
- **Filtering**: Quickly find services with powerful search capabilities (`/`).
- **Dev Mode**: Automatic filtering for common developer services (Docker, Postgres, etc.).

## Installation

### From Source (Go Developers)

If you have Go installed, you can install Vigilix directly:

```bash
go install github.com/LOVENISH87/vigilix/cmd/vigilix@latest
```

Ensure your `$GOPATH/bin` is in your `$PATH`.

### Manual Build

```bash
git clone https://github.com/LOVENISH87/vigilix.git
cd vigilix
go build -o vigilix cmd/vigilix/main.go
sudo mv vigilix /usr/local/bin/
```

## Usage

Since Vigilix interacts with systemd, it typically requires elevated privileges to manage system services:

```bash
sudo vigilix
```

### Key Bindings

| Key | Action |
| :--- | :--- |
| `↑` / `↓` / `j` / `k` | Navigate list |
| `/` | Search / Filter units |
| `Enter` | View logs for selected unit |
| `c` | View unit configuration |
| `s` | **Start** service |
| `x` | **Stop** service |
| `r` | **Restart** service |
| `d` | Toggle **Dev Mode** (filter common dev tools) |
| `q` | Quit |

## Technology Stack

- **Language**: Go (Golang)
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Styling**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **System Info**: [gopsutil](https://github.com/shirou/gopsutil)

## 📄 License

MIT License © 2026 
