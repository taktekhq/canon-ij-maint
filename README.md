# canon-ij-maint

Run print-head cleaning, **deep cleaning** and nozzle checks on Canon PIXMA inkjets from Linux,
without installing Canon's driver.

Canon's official Linux driver (`cnijfilter2`) only offers a regular cleaning and a nozzle check.
Deep cleaning is normally only available from Canon's Windows/macOS tools or the printer's own
buttons, and many owners never find the right button sequence. This tool sends the same commands
Canon's driver sends, plus the deep cleaning variant.

Website: <https://nizarmah.com/canon-ij-maint/>

## Install

Download the binary for your machine (no dependencies; `amd64` for most PCs, `arm64` for a
Raspberry Pi 4/5):

```bash
curl -fsSL -o canon-ij-maint https://github.com/nizarmah/canon-ij-maint/releases/latest/download/canon-ij-maint-linux-amd64
chmod +x canon-ij-maint && sudo mv canon-ij-maint /usr/local/bin/
```

Or build it with Go 1.22+:

```bash
go install github.com/nizarmah/canon-ij-maint/cmd/canon-ij-maint@latest
```

## Quick start

```bash
# list Canon printers this machine knows about
canon-ij-maint list

# regular cleaning
canon-ij-maint --queue YOUR_QUEUE clean

# deep cleaning (uses a lot more ink; let the printer rest ~10 minutes afterwards)
canon-ij-maint --queue YOUR_QUEUE clean --deep

# print the nozzle check pattern
canon-ij-maint --queue YOUR_QUEUE nozzle-check
```

The destination can be:

- `--queue NAME`: any CUPS queue pointing at the printer. The bytes go through `lp -o raw`, so
  the queue's driver doesn't matter.
- `--device /dev/usb/lp0`: write straight to the USB printer device (needs write access, usually
  the `lp` group).
- `--output FILE`: just save the bytes (`-` for stdout), e.g. to inspect them or send them another way.

If you give none and there is exactly one Canon CUPS queue, it uses that.

## Tested printers

| Printer | Regular clean | Deep clean | Nozzle check |
|---|---|---|---|
| PIXMA G3010 (USB) | works | works (`--group all`) | works |

All three commands are byte-for-byte identical to what Canon's own `cmdtocanonij3` filter sends.

On our G3010, magenta had stopped printing entirely, with plenty of ink in the tank. Two regular
cleanings did nothing. One deep cleaning, followed by a 10-minute rest, brought it back fully.

| Before | After one deep clean |
|---|---|
| ![Color test page before: magenta missing](docs/img/before.jpg) | ![Color test page after: magenta restored](docs/img/after.jpg) |

Unedited 150 dpi scans of the same test page, from the G3010's own scanner.

On the G3010, `--group color` deep cleaning is **ignored** (no sound, nothing happens). Use the
default `--group all`. `--group black` has not been tested.

Other recent PIXMA, MegaTank (G-series), TS and TR models that use Canon's `cnijfilter2` driver
probably speak the same protocol, but they're untested. If a command is ignored, the printer
simply stays silent; nothing harmful happens. Reports for other models are welcome.

## Use it from Go

The command builder is a small Go package, so other programs can send the same commands:

```go
import canonij "github.com/nizarmah/canon-ij-maint"

data, _ := canonij.Cleaning(canonij.Deep, canonij.All)
out, err := canonij.SendToQueue("Canon_G3010", "Deep cleaning", "", data)
```

[canon-home-printer](https://github.com/taktekhq/canon-home-printer), a family website for a
home G3010, uses it for its maintenance buttons.

## How it works

These printers accept "IVEC" XML commands on the print channel. Canon's `cmdtocanonij3` CUPS
filter frames each maintenance job as:

1. a `StartJob` message (`servicetype="maintenance"`),
2. as many zero bytes as that message is long,
3. the operation: `Cleaning` (`inkgroup` = all/black/color, `type` = regular/deep) or
   `TestPrint` (`type` = nozzle_check),
4. an `EndJob` message.

Canon's filter hard-codes `type=regular`. Changing it to `deep` triggers a deep cleaning.

The older Canon "BJL" commands (`@Cleaning=1ALL`, used by Gutenprint's `commandtocanon`) are
ignored by the G3010.

## Troubleshooting tips

- **Paper out:** after loading paper, press the printer's Black or Color button to resume.
- **One color missing after cleaning:** do a deep clean, wait 10 minutes, then print a test page.
  Repeat once if needed. If a color is still missing with ink in the tank, the print head is
  likely worn out (it's a replaceable part on MegaTank models).
- **Ubuntu auto-created queue:** on plug-in, Ubuntu may create a queue with a wrong driver
  (we saw an HP Designjet PostScript driver picked for a G3010). Delete it and add the printer
  with Gutenprint (`printer-driver-gutenprint`) or Canon's `cnijfilter2` driver.

## Not supported (yet)

- Ink flush / "system cleaning". Its command value is unknown.
- Reading ink levels or status back from the printer. That needs a two-way connection.

## Develop

```bash
go test ./...
```

`testdata/` holds the exact bytes that were sent to the G3010 and fixed it; the tests check
the output against them byte for byte.

## License and disclaimer

MIT. Use at your own risk: cleaning cycles use ink and fill the printer's waste-ink absorber.

Not affiliated with or endorsed by Canon Inc. "Canon", "PIXMA" and "MegaTank" are trademarks
of Canon Inc., used here only to say which printers this tool works with.
