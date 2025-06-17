package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

func CominServiceRestart() error {
	if runtime.GOOS == "darwin" {
		return cominServiceRestartDarwin()
	}
	return cominServiceRestartLinux()
}

func cominServiceRestartLinux() error {
	logrus.Infof("The comin.service unit file changed. Comin systemd service is now restarted...")
	logrus.Infof("Restarting the systemd comin.service: 'systemctl restart --no-block comin.service'")
	cmd := exec.Command("systemctl", "restart", "--no-block", "comin.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command 'systemctl restart --no-block comin.service' fails with %s", err)
	}
	return nil
}

func cominServiceRestartDarwin() error {
	logrus.Infof("Comin service configuration changed - will restart after deployment")
	
	// On Darwin, create a flag file that signals the main process should exit
	// after the current deployment completes. Launchd will automatically restart
	// the process due to KeepAlive=true, which is more reliable than self-restart
	
	restartFlagPath := "/var/lib/comin/restart-required"
	if err := os.WriteFile(restartFlagPath, []byte("restart after deployment"), 0644); err != nil {
		logrus.Warnf("Failed to create restart flag: %s", err)
		// Continue anyway - this is not critical for the deployment
	}
	
	logrus.Infof("Restart flag created - comin will exit after deployment and launchd will restart it")
	return nil
}

func FormatCommitMsg(msg string) string {
	split := strings.Split(msg, "\n")
	formatted := ""
	for i, s := range split {
		if i == len(split)-1 && s == "" {
			continue
		}
		if i == 0 {
			formatted += s
		} else {
			formatted += "\n    " + s
		}
	}
	return formatted
}

func ReadMachineId() (machineId string, err error) {
	if runtime.GOOS == "darwin" {
		return readMachineIdDarwin()
	}
	return readMachineIdLinux()
}

func readMachineIdLinux() (machineId string, err error) {
	machineIdBytes, err := os.ReadFile("/etc/machine-id")
	machineId = strings.TrimSuffix(string(machineIdBytes), "\n")
	if err != nil {
		return "", fmt.Errorf("can not read file '/etc/machine-id': %s", err)
	}
	return
}

func readMachineIdDarwin() (machineId string, err error) {
	cmd := exec.Command("/usr/sbin/system_profiler", "SPHardwareDataType")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get hardware UUID on macOS: %s", err)
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Hardware UUID:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				machineId = strings.TrimSpace(parts[1])
				return machineId, nil
			}
		}
	}
	return "", fmt.Errorf("could not find Hardware UUID in system_profiler output")
}
