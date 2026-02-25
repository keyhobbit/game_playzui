# Android APK Build Guide — Tien Len Mien Nam

This guide covers the complete environment setup to build a signed Android APK from the Godot 4.2 client.

---

## Table of Contents

1. [Prerequisites Overview](#1-prerequisites-overview)
2. [Install Java JDK 17](#2-install-java-jdk-17)
3. [Install Android SDK & NDK](#3-install-android-sdk--ndk)
4. [Install Godot 4.2](#4-install-godot-42)
5. [Install Android Export Templates](#5-install-android-export-templates)
6. [Configure Godot Android SDK Paths](#6-configure-godot-android-sdk-paths)
7. [Install the Android Build Template](#7-install-the-android-build-template)
8. [Create a Keystore (Release Signing)](#8-create-a-keystore-release-signing)
9. [Configure Export Presets](#9-configure-export-presets)
10. [Set Server IP / Config](#10-set-server-ip--config)
11. [Export the APK](#11-export-the-apk)
12. [Install on Device](#12-install-on-device)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Prerequisites Overview

| Tool | Required Version | Purpose |
|------|-----------------|---------|
| Java JDK | 17 (LTS) | Android build toolchain |
| Android SDK | API 34+ (SDK Tools 34+) | Android platform tools |
| Android NDK | r23c | Native code compilation |
| Godot Engine | 4.2.x | Project editor & exporter |
| Godot Export Templates | 4.2.x (matching engine) | Android template |

> **OS Note:** These instructions cover Linux (Ubuntu/Debian), macOS, and Windows (WSL2/PowerShell).

---

## 2. Install Java JDK 17

Godot 4.x requires **JDK 17**. JDK 21 also works. JDK 8 or 11 will cause Gradle errors.

### Linux (Ubuntu/Debian)
```bash
sudo apt update
sudo apt install -y openjdk-17-jdk

# Verify
java -version
# Expected: openjdk version "17.x.x"
```

### macOS (Homebrew)
```bash
brew install openjdk@17
sudo ln -sfn $(brew --prefix)/opt/openjdk@17/libexec/openjdk.jdk /Library/Java/JavaVirtualMachines/openjdk-17.jdk

# Verify
java -version
```

### Windows
Download the installer from https://adoptium.net/ (Eclipse Temurin JDK 17).  
During installation, check **"Add to PATH"** and **"Set JAVA_HOME"**.

---

## 3. Install Android SDK & NDK

### Option A — Android Studio (Recommended, includes GUI)

1. Download from https://developer.android.com/studio
2. Install and open Android Studio
3. Go to **SDK Manager** → **SDK Platforms** → install **Android 14 (API 34)**
4. Go to **SDK Manager** → **SDK Tools** → install:
   - Android SDK Build-Tools 34
   - Android SDK Command-line Tools (latest)
   - Android SDK Platform-Tools
   - NDK (Side by side) — install version **23.2.8568313** (r23c)

Default SDK location:
- Linux/macOS: `~/Android/Sdk`
- Windows: `C:\Users\<user>\AppData\Local\Android\Sdk`

### Option B — Command-line only (Linux/macOS)

```bash
# 1. Download SDK command-line tools
mkdir -p ~/Android/Sdk/cmdline-tools
cd ~/Android/Sdk/cmdline-tools
wget https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip
unzip commandlinetools-linux-11076708_latest.zip
mv cmdline-tools latest

# 2. Add to PATH
echo 'export ANDROID_HOME=$HOME/Android/Sdk' >> ~/.bashrc
echo 'export PATH=$PATH:$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/platform-tools' >> ~/.bashrc
source ~/.bashrc

# 3. Accept licenses
sdkmanager --licenses

# 4. Install required packages
sdkmanager "platform-tools" \
           "build-tools;34.0.0" \
           "platforms;android-34" \
           "ndk;23.2.8568313"

# Verify NDK path
ls ~/Android/Sdk/ndk/23.2.8568313/
```

---

## 4. Install Godot 4.2

The project targets Godot **4.2** (`config/features=PackedStringArray("4.2", "Mobile")`).

### Linux
```bash
# Download Godot 4.2-stable (standard, not Mono/C#)
wget https://github.com/godotengine/godot/releases/download/4.2-stable/Godot_v4.2-stable_linux.x86_64.zip
unzip Godot_v4.2-stable_linux.x86_64.zip
chmod +x Godot_v4.2-stable_linux.x86_64

# Optional: move to PATH
sudo mv Godot_v4.2-stable_linux.x86_64 /usr/local/bin/godot4
```

### macOS
Download the `.dmg` from https://godotengine.org/download/macos/ and drag to Applications.

### Windows
Download the `.exe` installer from https://godotengine.org/download/windows/.

> Use the **standard (non-Mono)** build — this project uses GDScript only.

---

## 5. Install Android Export Templates

Export templates must **exactly match** the Godot editor version.

1. In Godot: **Editor → Manage Export Templates**
2. Click **Download and Install**
3. Select version `4.2-stable` and click **Download**

Or install manually:
```bash
# Download templates
wget https://github.com/godotengine/godot/releases/download/4.2-stable/Godot_v4.2-stable_export_templates.tpz

# Godot will look for templates in:
# Linux:   ~/.local/share/godot/export_templates/4.2.stable/
# macOS:   ~/Library/Application Support/Godot/export_templates/4.2.stable/
# Windows: %APPDATA%\Godot\export_templates\4.2.stable\

mkdir -p ~/.local/share/godot/export_templates/4.2.stable/
cd ~/.local/share/godot/export_templates/4.2.stable/
unzip /path/to/Godot_v4.2-stable_export_templates.tpz --strip-components=1
```

---

## 6. Configure Godot Android SDK Paths

1. Open Godot Editor
2. Go to **Editor → Editor Settings → Export → Android**
3. Set the following paths:

| Setting | Value |
|---------|-------|
| Android Sdk Path | `~/Android/Sdk` (full path, no `~`) |
| Jdk Path | `/usr/lib/jvm/java-17-openjdk-amd64` (Linux) or output of `which java` parent |

**Linux quick reference:**
```bash
# Find JDK path
dirname $(dirname $(readlink -f $(which java)))
# Example output: /usr/lib/jvm/java-17-openjdk-amd64

# Find SDK path
echo $ANDROID_HOME
# Example output: /home/<user>/Android/Sdk
```

---

## 7. Install the Android Build Template

This step generates the Gradle project inside the client folder.

1. Open the project in Godot: `File → Open Project` → select `client/project.godot`
2. Go to **Project → Install Android Build Template**
3. A `client/android/` directory will be created — this is the Gradle project

```bash
# After installation the structure will include:
client/
└── android/
    ├── build/
    │   └── src/
    └── build.gradle
```

> Re-run this step whenever you upgrade the Godot engine version.

---

## 8. Create a Keystore (Release Signing)

A keystore is required for release APKs. Debug builds use Godot's built-in debug key.

```bash
# Create a release keystore (run once, keep the .keystore file safe)
keytool -genkey -v \
  -keystore ~/tienlen-release.keystore \
  -alias tienlen \
  -keyalg RSA \
  -keysize 2048 \
  -validity 10000

# You will be prompted for:
# - Keystore password
# - Key password
# - Your name / organization / location (can be anything for dev)
```

> **Important:** Back up `tienlen-release.keystore` and both passwords. Losing the keystore means you cannot update the app on Play Store.

---

## 9. Configure Export Presets

1. In Godot: **Project → Export**
2. Click **Add** → select **Android**
3. Configure the preset:

**Presets tab:**
| Field | Value |
|-------|-------|
| Name | `Android Release` |
| Export Path | `../tienlen.apk` (relative to `client/`) |
| Use Gradle Build | ✅ Enabled |

**Options → Keystore tab (Release):**
| Field | Value |
|-------|-------|
| Release Keystore | `/home/<user>/tienlen-release.keystore` |
| Release User | `tienlen` (alias used in keytool) |
| Release Password | *(your keystore password)* |

**Options → Package tab:**
| Field | Value |
|-------|-------|
| Unique Name | `com.yourstudio.tienlen` |
| Min SDK | 24 |
| Target SDK | 34 |

**Permissions tab** — enable:
- `INTERNET` ✅ (required for WebSocket connection)
- `ACCESS_NETWORK_STATE` ✅

> The `export_presets.cfg` file will be auto-saved in `client/`. Commit it (but **do not** commit the keystore or passwords — use environment variables in CI).

---

## 10. Set Server IP / Config

Before exporting, update the server address in:

```
client/scripts/autoload/NetworkConfig.gd
```

```gdscript
# For production (VPS/server):
const SERVER_HOST = "your.server.ip.or.domain"
const SERVER_PORT = 8700

# For local testing on the same WiFi:
const SERVER_HOST = "192.168.x.x"   # your computer's LAN IP
const SERVER_PORT = 8700
```

Find your LAN IP with:
```bash
hostname -I | awk '{print $1}'
```

---

## 11. Export the APK

### Via Godot Editor (GUI)

1. **Project → Export**
2. Select **Android Release** preset
3. Click **Export Project**
4. Choose output path and file name (e.g. `tienlen.apk`)
5. Click **Save**

### Via Command Line (Headless / CI)

```bash
cd /path/to/game_playzui/client

# Debug APK (quick test)
godot4 --headless --export-debug "Android Release" ../tienlen-debug.apk

# Release APK (signed, for distribution)
godot4 --headless --export-release "Android Release" ../tienlen.apk
```

> The preset name (`"Android Release"`) must match exactly what you named it in step 9.

---

## 12. Install on Device

### Via ADB (USB cable)

```bash
# Enable USB Debugging on your phone:
# Settings → About Phone → tap "Build Number" 7 times → Developer Options → USB Debugging ON

# Connect phone and verify it's detected
adb devices

# Install
adb install -r tienlen.apk

# Or for release APK that replaces debug:
adb install -r -d tienlen.apk
```

### Via Web Browser (same WiFi)

```bash
# Serve the APK for download
python3 -m http.server 9000

# On your phone browser:
# http://<your-computer-ip>:9000/tienlen.apk
# Allow "Install from unknown sources" on the phone
```

### Via Quick Web Test (no APK needed)

```bash
# Start backend
docker compose up -d

# Serve web test client
cd /path/to/game_playzui
python3 -m http.server 9000

# Open on phone: http://<your-computer-ip>:9000/tools/test_client.html
```

---

## 13. Troubleshooting

### `JAVA_HOME is not set` or wrong JDK version
```bash
export JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
export PATH=$JAVA_HOME/bin:$PATH
# Add to ~/.bashrc to persist
```

### `NDK not found` error in Godot
- Verify NDK r23c is installed: `ls ~/Android/Sdk/ndk/`
- In Godot Editor Settings, ensure **Android Sdk Path** points to the `Sdk` folder (not `Sdk/ndk`)

### Gradle build fails: `Could not determine java version`
- Godot's embedded Gradle requires JDK 17. Run `java -version` and confirm it's 17

### `adb: error: failed to install` — INSTALL_FAILED_UPDATE_INCOMPATIBLE
- The debug and release APKs have different signatures. Uninstall the old version first:
```bash
adb uninstall com.yourstudio.tienlen
adb install tienlen.apk
```

### WebSocket connects but game doesn't work
- Confirm `SERVER_HOST` in `NetworkConfig.gd` matches your running backend IP
- Check backend is up: `curl http://<server-ip>:8700/health`
- Ensure port `8700` is open in any firewall: `sudo ufw allow 8700`

### Export templates not found
- Confirm templates version matches editor: **Editor → About → Version** must equal the template folder name
- Reinstall via **Editor → Manage Export Templates**
