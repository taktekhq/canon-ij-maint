// Command canon-ij-maint runs cleaning and nozzle checks on Canon PIXMA inkjets without Canon's driver.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	canonij "github.com/nizarmah/canon-ij-maint"
)

const usage = `canon-ij-maint: clean Canon PIXMA print heads and print nozzle checks from Linux.

Usage:
  canon-ij-maint [destination] clean [--deep] [--group all|black|color]
  canon-ij-maint [destination] nozzle-check
  canon-ij-maint list

Destination (default: the only Canon CUPS queue, or the only /dev/usb/lp* device):
  --queue NAME     a CUPS queue for the printer (sent with lp -o raw)
  --device PATH    the USB printer device, e.g. /dev/usb/lp0 (needs write access)
  --output FILE    write the bytes to a file instead ("-" for stdout)

The G3010 ignores --group color; use the default, all.
`

type dest struct{ queue, device, output string }

func (d *dest) register(fs *flag.FlagSet) {
	fs.StringVar(&d.queue, "queue", d.queue, "CUPS queue name")
	fs.StringVar(&d.device, "device", d.device, "USB printer device")
	fs.StringVar(&d.output, "output", d.output, `write bytes to a file ("-" = stdout)`)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	var d dest
	top := flag.NewFlagSet("canon-ij-maint", flag.ContinueOnError)
	top.SetOutput(stderr)
	top.Usage = func() { fmt.Fprint(stderr, usage) }
	d.register(top)
	if err := top.Parse(args); err != nil {
		return 2
	}
	if top.NArg() == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd, rest := top.Arg(0), top.Args()[1:]

	sub := flag.NewFlagSet(cmd, flag.ContinueOnError)
	sub.SetOutput(stderr)
	sub.Usage = func() { fmt.Fprint(stderr, usage) }
	d.register(sub)

	var data []byte
	var title string
	switch cmd {
	case "list":
		if err := sub.Parse(rest); err != nil {
			return 2
		}
		for _, q := range canonij.FindQueues() {
			fmt.Fprintf(stdout, "queue   %s\n", q)
		}
		for _, dev := range canonij.FindDevices() {
			fmt.Fprintf(stdout, "device  %s\n", dev)
		}
		return 0
	case "clean":
		deep := sub.Bool("deep", false, "deep cleaning (uses much more ink)")
		group := sub.String("group", "all", "ink group: all, black or color")
		if err := sub.Parse(rest); err != nil {
			return 2
		}
		kind, label := canonij.Regular, "Regular"
		if *deep {
			kind, label = canonij.Deep, "Deep"
		}
		var err error
		if data, err = canonij.Cleaning(kind, canonij.Group(*group)); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
		title = fmt.Sprintf("%s cleaning (%s)", label, *group)
	case "nozzle-check":
		if err := sub.Parse(rest); err != nil {
			return 2
		}
		data, title = canonij.NozzleCheck(), "Nozzle check"
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}

	if d.queue == "" && d.device == "" && d.output == "" {
		queues := canonij.FindQueues()
		devices := canonij.FindDevices()
		switch {
		case len(queues) == 1:
			d.queue = queues[0]
			fmt.Fprintf(stderr, "using CUPS queue %s\n", d.queue)
		case len(queues) == 0 && len(devices) == 1:
			d.device = devices[0]
			fmt.Fprintf(stderr, "using device %s\n", d.device)
		default:
			found := "none"
			if len(queues) > 0 {
				found = strings.Join(queues, ", ")
			}
			fmt.Fprintf(stderr, "Pick a destination with --queue, --device or --output (Canon queues found: %s).\n", found)
			return 1
		}
	}

	switch {
	case d.output == "-":
		if _, err := stdout.Write(data); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	case d.output != "":
		if err := os.WriteFile(d.output, data, 0o644); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintf(stderr, "wrote %d bytes to %s\n", len(data), d.output)
		return 0
	case d.device != "":
		if err := canonij.WriteDevice(d.device, data); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintf(stderr, "sent %d bytes to %s\n", len(data), d.device)
	default:
		msg, err := canonij.SendToQueue(d.queue, title, "", data)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stderr, msg)
	}
	if cmd == "clean" {
		fmt.Fprintln(stderr, "The printer should start whirring within a few seconds. If it stays silent, it ignored the command.")
	}
	return 0
}
