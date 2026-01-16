package main

import (
	"flag"
	"fmt"
	"os/exec"
)

type CommandRunner func() error

func main() {
	distro := flag.String("distro", "", "Linux distribution")
	flag.Parse()

	runners := map[string]CommandRunner{
		// Debian / Ubuntu family (apt)
		"ubuntu":      runUbuntuCommands,
		"debian":      runUbuntuCommands,
		"linuxmint":   runUbuntuCommands,
		"mint":        runUbuntuCommands,
		"pop":         runUbuntuCommands,
		"popos":       runUbuntuCommands,
		"kali":        runUbuntuCommands,
		"elementary":  runUbuntuCommands,
		"raspbian":    runUbuntuCommands,
		"raspberrypi": runUbuntuCommands,

		// Red Hat family (dnf)
		"fedora":       runFedoraCommands,
		"rhel":         runFedoraCommands,
		"redhat":       runFedoraCommands,
		"centos":       runFedoraCommands,
		"centosstream": runFedoraCommands,
		"rocky":        runFedoraCommands,
		"rockylinux":   runFedoraCommands,
		"alma":         runFedoraCommands,
		"almalinux":    runFedoraCommands,

		// Arch family (pacman)
		"arch":        runArchCommands,
		"archlinux":   runArchCommands,
		"manjaro":     runArchCommands,
		"endeavouros": runArchCommands,

		// SUSE family (zypper)
		"suse":         runSUSECommands,
		"opensuse":     runSUSECommands,
		"opensuseleap": runSUSECommands,
		"tumbleweed":   runSUSECommands,
		"gentoo":       runGentooCommands,

		// Void Linux (xbps)
		"void":      runVoidCommands,
		"voidlinux": runVoidCommands,

		// macOS (brew)
		"darwin": runMacCommands,
		"macos":  runMacCommands,
		"mac":    runMacCommands,

		// Windows (winget / choco)
		"windows": runWindowsCommands,
		"win":     runWindowsCommands,
	}

	runner, exists := runners[*distro]
	if !exists {
		fmt.Printf("Unknown distro: %s\n", *distro)
		return
	}

	if err := runner(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func runUbuntuCommands() error {
	commands := [][]string{
		{"apt", "update"},
		{"apt", "install", "-y", "curl"},
		{"apt", "install", "-y", "git"},
		{"apt", "upgrade", "-y"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runFedoraCommands() error {
	commands := [][]string{
		{"dnf", "update", "-y"},
		{"dnf", "install", "-y", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runSUSECommands() error {
	commands := [][]string{
		{"zypper", "update", "-y"},
		{"zypper", "install", "-y", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runArchCommands() error {
	commands := [][]string{
		{"pacman", "-Syu", "--noconfirm"},
		{"pacman", "-S", "--noconfirm", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runGentooCommands() error {
	commands := [][]string{
		{"emerge", "--sync"},
		{"emerge", "--ask=n", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runVoidCommands() error {
	commands := [][]string{
		{"xbps-install", "-Suy"},
		{"xbps-install", "-y", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runMacCommands() error {
	commands := [][]string{
		{"brew", "update"},
		{"brew", "install", "curl", "git"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}

func runWindowsCommands() error {
	commands := [][]string{
		{"winget", "upgrade", "--all", "--silent"},
		{"winget", "install", "--id", "Git.Git", "--silent"},
		{"winget", "install", "--id", "Curl.Curl", "--silent"},
	}

	for _, cmd := range commands {
		fmt.Printf("Running: %v\n", cmd)
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", cmd, err)
		}
	}
	return nil
}
