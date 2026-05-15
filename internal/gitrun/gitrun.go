package gitrun

import (
	"os"
	"os/exec"

	"github.com/frimpsss/gzy/internal/config"
)

type Command struct {
	Args []string
	Env  map[string]string
}

type Runner struct {
	GitPath string
	GZYPath string
}

func BuildCommand(account config.Account, gitArgs []string, gzyPath string) Command {
	args := []string{
		"-c", "user.name=" + account.Name,
		"-c", "user.email=" + account.Email,
	}
	if gzyPath != "" {
		args = append(args,
			"-c", "credential.https://github.com.helper=",
			"-c", "credential.https://github.com.helper=!"+gzyPath+" credential "+account.Alias,
		)
	}
	args = append(args, gitArgs...)
	env := map[string]string{
		"GIT_SSH_COMMAND": "ssh -i " + account.PrivateKey + " -o IdentitiesOnly=yes",
	}
	return Command{Args: args, Env: env}
}

func (r Runner) Run(account config.Account, gitArgs []string) (int, error) {
	gitPath := r.GitPath
	if gitPath == "" {
		gitPath = "git"
	}
	built := BuildCommand(account, gitArgs, r.GZYPath)
	cmd := exec.Command(gitPath, built.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	for key, value := range built.Env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode(), nil
	}
	return 1, err
}
