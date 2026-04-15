# ObsidianSync — Design Spec
**Date:** 2026-04-15

## Overview

A compiled Windows `.exe` that performs a two-way sync between a local Obsidian vault and an iCloud Drive folder. Run manually on demand. No background daemon or watch mode.

- **Source:** `C:\Ariq\Jade Chamber\Obsidian`
- **Destination:** `C:\Users\rifar\iCloudDrive\iCloud~md~obsidian\Jade Chamber`

---

## File & Folder Layout

```
C:\Users\rifar\ObsidianSync\
    obsync.exe          ← compiled binary
    sync.toml           ← config file (paths + ignore patterns)
    sync-state.json     ← snapshot of last sync state (auto-managed)
    sync.log            ← running log of all sync operations

C:\Users\rifar\Desktop\
    ObsidianSync.lnk    ← shortcut pointing to obsync.exe
```

### sync.toml

```toml
[paths]
source      = 'C:\Ariq\Jade Chamber\Obsidian'
destination = 'C:\Users\rifar\iCloudDrive\iCloud~md~obsidian\Jade Chamber'

[ignore]
patterns = [
    ".stfolder",
    ".stignore",
    ".stversions",
    ".data",
]
```

Ignore patterns apply to both file names and directory names. A matched directory is skipped entirely (no traversal into it).

---

## Sync Logic

One sync pass per run. The program exits when done.

### Algorithm

1. Load `sync.toml`. If missing, print error and exit.
2. Load `sync-state.json` (snapshot of files from last sync). If missing, treat as first run (no deletions propagated).
3. Walk source and destination directories, building file lists. Exclude any path matching an ignore pattern.
4. For each file, determine action using last-modified timestamps and snapshot:

| File present in | Action |
|---|---|
| Source only, not in snapshot | Copy source → destination |
| Destination only, not in snapshot | Copy destination → source |
| Source only, was in snapshot | Delete from destination (was deleted at source) |
| Destination only, was in snapshot | Delete from source (was deleted at destination) |
| Both sides, same mtime | Skip (unchanged) |
| Both sides, source newer | Copy source → destination |
| Both sides, destination newer | Copy destination → source |

5. If any deletions are pending, show confirmation prompt (see UX section). If user selects N, cancel all operations and exit.
6. Execute all operations, logging each one.
7. Update `sync-state.json` with the new file list and mtimes.

### Conflict Resolution

"Last modified wins." Since the user does not edit both sides simultaneously, true conflicts are not expected. No special conflict handling beyond timestamp comparison.

---

## UX & Console Output

### When deletions are pending

```
ObsidianSync v1.0
Source:      C:\Ariq\Jade Chamber\Obsidian
Destination: C:\Users\rifar\iCloudDrive\iCloud~md~obsidian\Jade Chamber

Scanning...

⚠ Warning: 2 file(s) will be deleted:
    00-Inbox/old-note.md
    10-Journal/2026-03-01.md

Proceed with sync? (includes 3 copies, 2 updates, 2 deletions) [Y/N]: _
```

- **Y or y** — proceed with all operations (copies, updates, and deletions)
- **N or n** — print `Sync cancelled. No changes made.` and exit; nothing is touched
- Input is case-insensitive.

### When no deletions are pending

No prompt. Sync runs immediately.

### After sync

```
[→] Copied   10-Journal/2026-04-15.md   (source → dest)
[←] Copied   30-Entities/Ariq.md        (dest → source)
[✕] Deleted  00-Inbox/old-note.md       (removed from both)

Done. 2 copied, 2 updated, 1 deleted, 47 unchanged.
Press any key to exit...
```

The console window stays open until a key is pressed.

### Errors

- If a file can't be copied or deleted (locked, permission denied), log the error and continue — do not abort the entire sync.
- All errors are written to `sync.log` with timestamps.

---

## Logging

`sync.log` next to the `.exe` records every run:

```
[2026-04-15 14:32:01] Sync started
[2026-04-15 14:32:01] Copied: 10-Journal/2026-04-15.md (source → dest)
[2026-04-15 14:32:01] Deleted: 00-Inbox/old-note.md
[2026-04-15 14:32:02] Sync complete. 2 copied, 1 deleted, 47 unchanged.
```

---

## Implementation Language

**Go** — compiles to a single native `.exe` with no runtime dependencies. Small binary (~5MB), instant startup.

### Key dependencies

- `github.com/BurntSushi/toml` — TOML config parsing
- Standard library only for everything else (`os`, `path/filepath`, `encoding/json`, `time`)
