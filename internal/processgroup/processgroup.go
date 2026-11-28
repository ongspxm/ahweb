package processgroup

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

func Start(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	return cmd.Start()
}

func Stop(cmd *exec.Cmd, done <-chan struct{}, grace time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		if done != nil {
			<-done
		}
		return nil
	}
	if err := Interrupt(cmd); err != nil {
		return err
	}
	if done == nil {
		return Kill(cmd)
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
	}
	if err := Kill(cmd); err != nil {
		return err
	}
	<-done
	return nil
}

func Kill(cmd *exec.Cmd) error {
	return signal(cmd, syscall.SIGKILL)
}

func Interrupt(cmd *exec.Cmd) error {
	return signal(cmd, syscall.SIGINT)
}

func signal(cmd *exec.Cmd, sig syscall.Signal) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}
