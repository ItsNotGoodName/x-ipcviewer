package mpv

import (
	"os"
	"os/exec"

	"github.com/ItsNotGoodName/mpvipc"
	"github.com/ItsNotGoodName/x-ipc-viewer/closer"
)

func cmdCloser(cmd *exec.Cmd) closer.Closer {
	return func() error {
		return cmd.Process.Kill()
	}
}

func connectionCloser(conn *mpvipc.Connection, socketPath string) closer.Closer {
	return func() error {
		if err := conn.Close(); err != nil {
			return err
		}
		return os.Remove(socketPath)
	}
}
