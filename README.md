# Unreal Free Assets Monitor

A lightweight Windows system tray application that keeps you informed about free asset drops on [FAB](https://fab.com) and the Unreal Marketplace.

## Why This Exists

Game development is an incredible creative journey. Whether you're building your first prototype or shipping your tenth title, having access to quality assets can accelerate your workflow and spark new ideas. Epic Games regularly offers free asset packs through FAB - but it's easy to miss these limited-time offers.

This tool runs quietly in your system tray, checking for new free assets every hour and notifying you when fresh content becomes available. Never miss another free asset drop.

## Features

- **System Tray App** - Runs silently in the background
- **Hourly Checks** - Automatically monitors for new free assets
- **Windows Notifications** - Get notified when new free assets appear
- **Native UI** - Beautiful dark-themed interface with Unreal orange accents
- **Search & Filter** - Quickly find assets by name
- **Latest News** - Stay updated on Unreal Engine releases and marketplace events

## Screenshots

The app lives in your system tray. Right-click to access options or view assets.

## Installation

1. Download the latest installer from [Releases](../../releases)
2. Run `UnrealFreeAssetsSetup-1.0.0.exe`
3. The app will start automatically and appear in your system tray

## Building from Source

Requires Go 1.21+ and a C compiler (for Fyne GUI).

```bash
git clone https://github.com/yourusername/unreal-free-assets.git
cd unreal-free-assets
go build -ldflags="-H windowsgui -s -w" -o unreal-free-assets.exe .
```

## How It Works

The app monitors [unrealsource.com/dispatch](https://unrealsource.com/dispatch/) for announcements about free FAB assets. When Epic drops a new batch (typically every two weeks), you'll get a notification with direct links to claim them.

## Support

If you find this useful, consider [buying me a coffee](https://buymeacoffee.com/qvark).

## License

MIT License - Feel free to use, modify, and distribute.

---

Built for the Unreal Engine community. Happy developing!
