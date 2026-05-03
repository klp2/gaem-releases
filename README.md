# gaem ⚔

> _Description coming soon._
> <!-- TODO: drop in 2-3 sentences in your voice + maybe an animated dungeon-crawl GIF -->

Pre-built cross-platform binaries for **gaem**, a terminal roguelike.

## Download

Grab the latest archive from the **[Releases](https://github.com/klp2/gaem-releases/releases/latest)** page.

| OS | Architecture | File |
|----|--------------|------|
| Linux | x86-64 | `gaem-vX.Y.Z-linux-amd64.tar.gz` |
| Linux | ARM64 | `gaem-vX.Y.Z-linux-arm64.tar.gz` |
| macOS | Intel | `gaem-vX.Y.Z-darwin-amd64.tar.gz` |
| macOS | Apple Silicon | `gaem-vX.Y.Z-darwin-arm64.tar.gz` |
| Windows | x86-64 | `gaem-vX.Y.Z-windows-amd64.zip` |

## Install & Run

### Linux

```sh
tar xf gaem-vX.Y.Z-linux-*.tar.gz
cd gaem-vX.Y.Z-linux-*
chmod +x gaem
./gaem
```

### macOS

```sh
tar xf gaem-vX.Y.Z-darwin-*.tar.gz
cd gaem-vX.Y.Z-darwin-*
chmod +x gaem
./gaem
```

The binary is unsigned, so on first launch macOS Gatekeeper will refuse to run it. Either right-click `gaem` in Finder and choose **Open** (then **Open** again in the dialog), or, after the first failed launch, visit **System Settings → Privacy & Security → "Open Anyway"**. Subsequent runs from the terminal work normally.

### Windows

Extract the zip, then in **Windows Terminal** (1.0 or newer):

```powershell
cd gaem-vX.Y.Z-windows-amd64
.\gaem.exe
```

Windows Defender SmartScreen may show "Windows protected your PC" on first run. Click **More info** → **Run anyway**.

## Requirements

- A 256-color terminal at least ~80 columns wide.
- **Linux:** any modern terminal emulator.
- **macOS:** Terminal.app or iTerm2.
- **Windows:** Windows Terminal (1.0+) recommended; the legacy `cmd.exe` host may not render the Unicode glyphs correctly.

## Verifying the version

```sh
./gaem --version
```

Should print the tag the binary was built from (e.g. `gaem v0.1.0`).
