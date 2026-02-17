package systemd

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

type Unit struct {
	Name        string
	LoadState   string
	ActiveState string
	SubState    string
	Description string
}

// ListUnits returns a list of all systemd units.
func ListUnits() ([]Unit, error) {
	// We use --no-legend and --no-pager for easier parsing
	cmd := exec.Command("systemctl", "list-units", "--all", "--no-legend", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseUnits(string(output)), nil
}

func parseUnits(output string) []Unit {
	var units []Unit
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// systemctl output format varies, but usually:
		// UNIT LOAD ACTIVE SUB DESCRIPTION
		// We'll take a simplified approach for now
		unit := Unit{
			Name:        fields[0],
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
			Description: strings.Join(fields[4:], " "),
		}
		units = append(units, unit)
	}
	return units
}

func StartUnit(name string) error {
	return exec.Command("systemctl", "start", name).Run()
}

func StopUnit(name string) error {
	return exec.Command("systemctl", "stop", name).Run()
}

func RestartUnit(name string) error {
	return exec.Command("systemctl", "restart", name).Run()
}

func EnableUnit(name string) error {
	return exec.Command("systemctl", "enable", name).Run()
}

func DisableUnit(name string) error {
	return exec.Command("systemctl", "disable", name).Run()
}

func GetLogs(name string) (string, error) {
	// journalctl -u name -n 100 --no-pager
	cmd := exec.Command("journalctl", "-u", name, "-n", "100", "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func StreamLogs(ctx context.Context, name string, out chan<- string) error {
	cmd := exec.CommandContext(ctx, "journalctl", "-f", "-u", name, "--no-pager")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()
		select {
		case <-ctx.Done():
			return nil
		case out <- text:
		}
	}
	return cmd.Wait()
}

func GetUnitFileContent(name string) (string, error) {
	cmd := exec.Command("systemctl", "cat", name, "--no-pager")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
