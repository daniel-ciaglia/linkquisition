package resources

import _ "embed"

//go:embed Icon.png
var LinkquisitionIconBytes []byte

// SystemdUserServiceUnit is the content of the linkquisition.service user unit.
// Install to ~/.config/systemd/user/ or /usr/lib/systemd/user/ and run:
//
//	systemctl --user enable --now linkquisition.service
//
// When using the systemd unit, set daemon.autoStart to false in config.json to
// avoid two processes competing to become the primary GApplication instance.
//
//go:embed linkquisition.service
var SystemdUserServiceUnit []byte
