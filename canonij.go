// Package canonij builds the maintenance commands Canon PIXMA inkjets accept, and sends them.
//
// Newer Canon PIXMA printers (for example the G3010 MegaTank) take maintenance commands as small
// "IVEC" XML messages. Canon's Linux driver only exposes a regular cleaning and a nozzle check.
// This package builds the same bytes Canon's driver sends, including a deep cleaning.
//
// Not affiliated with or endorsed by Canon Inc. "Canon" and "PIXMA" are trademarks of Canon Inc.
package canonij

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Group is the set of inks a cleaning works on.
type Group string

// Ink groups. The G3010 ignores Color and cleans nothing; use All.
const (
	All   Group = "all"
	Black Group = "black"
	Color Group = "color"
)

// Groups lists the valid ink groups.
var Groups = []Group{All, Black, Color}

// Kind is how hard a cleaning works.
type Kind string

// Cleaning kinds. Deep uses much more ink.
const (
	Regular Kind = "regular"
	Deep    Kind = "deep"
)

// DefaultJobID is the job id Canon's driver uses.
const DefaultJobID = "00000001"

const ns = `xmlns:ivec="http://www.canon.com/ns/cmd/2008/07/common/"`

func startJob(jobID string) string {
	return `<?xml version="1.0" encoding="utf-8" ?>` +
		`<cmd ` + ns + `><ivec:contents><ivec:operation>StartJob</ivec:operation>` +
		`<ivec:param_set servicetype="maintenance">` +
		`<ivec:jobID>` + jobID + `</ivec:jobID><ivec:bidi>0</ivec:bidi>` +
		`<ivec:jobname/><ivec:username/><ivec:computername/>` +
		`<ivec:job_description><![CDATA[1]]></ivec:job_description>` +
		`</ivec:param_set></ivec:contents></cmd>`
}

func endJob(jobID string) string {
	return `<?xml version="1.0" encoding="utf-8" ?>` +
		`<cmd ` + ns + `><ivec:contents><ivec:operation>EndJob</ivec:operation>` +
		`<ivec:param_set servicetype="maintenance">` +
		`<ivec:jobID>` + jobID + `</ivec:jobID>` +
		`</ivec:param_set></ivec:contents></cmd>`
}

func operation(name, params, jobID string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<cmd ` + ns + `><ivec:contents><ivec:operation>` + name + `</ivec:operation>` +
		`<ivec:param_set servicetype="maintenance">` + params +
		`<ivec:jobID>` + jobID + `</ivec:jobID>` +
		`</ivec:param_set></ivec:contents></cmd>`
}

// Wrap frames one operation the way Canon's cmdtocanonij filter does:
// StartJob, then as many NUL bytes as StartJob is long, then the operation, then EndJob.
func Wrap(op, jobID string) []byte {
	start := startJob(jobID)
	var b bytes.Buffer
	b.WriteString(start)
	b.Write(make([]byte, len(start)))
	b.WriteString(op)
	b.WriteString(endJob(jobID))
	return b.Bytes()
}

// Cleaning returns the bytes for a print head cleaning.
func Cleaning(kind Kind, group Group) ([]byte, error) {
	if kind != Regular && kind != Deep {
		return nil, fmt.Errorf("cleaning kind must be %q or %q, not %q", Regular, Deep, kind)
	}
	if group != All && group != Black && group != Color {
		return nil, fmt.Errorf("ink group must be all, black or color, not %q", group)
	}
	params := `<ivec:inkgroup>` + string(group) + `</ivec:inkgroup><ivec:type>` + string(kind) + `</ivec:type>`
	return Wrap(operation("Cleaning", params, DefaultJobID), DefaultJobID), nil
}

// NozzleCheck returns the bytes that print the nozzle check pattern.
func NozzleCheck() []byte {
	return Wrap(operation("TestPrint", `<ivec:type>nozzle_check</ivec:type>`, DefaultJobID), DefaultJobID)
}

// ParseQueues returns the queue names in `lpstat -v` output whose device URI is a Canon printer.
func ParseQueues(lpstatV string) []string {
	var queues []string
	for _, line := range strings.Split(lpstatV, "\n") {
		rest, ok := strings.CutPrefix(line, "device for ")
		if !ok || !strings.Contains(strings.ToLower(line), "canon") {
			continue
		}
		name, _, _ := strings.Cut(rest, ":")
		queues = append(queues, name)
	}
	return queues
}

// FindQueues lists the CUPS queues for Canon printers. It returns nothing if CUPS isn't there.
func FindQueues() []string {
	out, err := exec.Command("lpstat", "-v").Output()
	if err != nil {
		return nil
	}
	return ParseQueues(string(out))
}

// FindDevices lists USB printer devices such as /dev/usb/lp0.
func FindDevices() []string {
	devs, _ := filepath.Glob("/dev/usb/lp*")
	sort.Strings(devs)
	return devs
}

// SendToQueue submits data as a raw job to a CUPS queue with `lp -o raw`.
// user, if set, is the name the job is recorded under.
// It returns what lp printed, which includes the job id.
func SendToQueue(queue, title, user string, data []byte) (string, error) {
	args := []string{"-d", queue, "-o", "raw", "-t", title}
	if user != "" {
		args = append([]string{"-U", user}, args...)
	}
	cmd := exec.Command("lp", args...)
	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// WriteDevice writes data straight to a USB printer device.
func WriteDevice(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
