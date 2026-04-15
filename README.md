# SimplySync

I made this tool to perform two-way sync between a local [Obsidian](https://obsidian.md) vault and an iCloud Drive folder. Run it on demand — no background daemon, no watch mode.

![icon](simplysync.ico)

---

## How It Works

Each run compares both folders and syncs them using a simple rule: **last modified wins**.

A snapshot (`sync-state.json`) is saved after every successful sync to track deletions. If a file disappears from one side since the last run, it gets deleted from the other side too — but only after you confirm.

| Situation | Action |
|---|---|
| File only in source, not in snapshot | Copy → destination |
| File only in destination, not in snapshot | Copy → source |
| File only in source, was in snapshot | Delete from destination |
| File only in destination, was in snapshot | Delete from source |
| Both sides, same modification time | Skip |
| Both sides, source newer | Copy → destination |
| Both sides, destination newer | Copy → source |

---

## Files

```
SimplySync/
├── simplysync.exe      ← run this to sync
├── sync.toml           ← configure your paths and ignore patterns
├── sync-state.json     ← auto-managed snapshot (do not edit)
└── sync.log            ← running log of all sync operations
```

---

## Configuration

Create, Copy, and Edit this text as `sync.toml` to set your source and destination paths, and any folder/file names to ignore:

```toml
[paths]
source      = 'C:\path\to\your\ObsidianVault'
destination = 'C:\Users\you\iCloudDrive\iCloud~md~obsidian\YourVault'

[ignore]
patterns = [
    "*.py"
    ".folder",
]
```

Ignore patterns match against individual path components (folder or file names). A matched directory is skipped entirely — nothing inside it is synced.

---

## Usage

Double-click `simplysync.exe` (or the desktop shortcut). The console window shows what's happening:

```
SimplySync v1.0
Source:      C:\path\to\your\ObsidianVault
Destination: C:\Users\you\iCloudDrive\iCloud~md~obsidian\YourVault

Scanning...

⚠ Warning: 2 file(s) will be deleted:
    00-Inbox/old-note.md
    10-Journal/2026-03-01.md

Proceed with sync? (includes 3 copies, 2 updates, 2 deletions) [Y/N]: y

[→] Copied   10-Journal/2026-04-15.md   (source → dest)
[←] Copied   30-Entities/Ariq.md        (dest → source)
[✕] Deleted  00-Inbox/old-note.md

Done. 1 copied, 1 updated, 1 deleted, 47 unchanged.

Press any key to exit...
```

- If **no deletions** are pending, sync runs immediately without prompting.
- **Y / y** — proceed with all operations.
- **N / n** — cancel; nothing is touched.

Errors (locked files, permission denied) are logged and skipped — the rest of the sync continues.

---

## Building from Source

Requires [Go 1.21+](https://go.dev/dl/).

```bash
git clone https://github.com/ariq-prabaswara/SimplySync.git
cd SimplySync
go mod download
go build -o simplysync.exe .
```

### Desktop shortcut (optional)

```powershell
$ws = New-Object -ComObject WScript.Shell
$s = $ws.CreateShortcut("$env:USERPROFILE\Desktop\SimplySync.lnk")
$s.TargetPath = "$PWD\simplysync.exe"
$s.WorkingDirectory = "$PWD"
$s.IconLocation = "$PWD\simplysync.ico,0"
$s.Save()
```

---

## Dependencies

- [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) — TOML config parsing
- Go standard library for everything else

---

## License

MIT
