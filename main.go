package main

import (
	"fmt"
	"github.com/google/shlex"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	log.SetPrefix("repokey: ")
	log.SetFlags(0)
	sshParams := os.Args[1:]
	if len(sshParams) == 0 {
		log.Fatalf("Usage: GIT_SSH_COMMAND=%s git clone ...", os.Args[0])
	}
	log.Printf("ssh params: %#v", sshParams)
	remoteCmd := sshParams[len(sshParams)-1]
	remoteCmdParts, err := shlex.Split(remoteCmd)
	if err != nil {
		panic(fmt.Errorf("couldn't parse ssh remote cmd %q: %+v", remoteCmd, err))
	}
	if len(remoteCmdParts) < 2 {
		panic(fmt.Errorf("don't know how to parse ssh remote cmd %q for repo path", remoteCmd))
	}
	repoPath := remoteCmdParts[len(remoteCmdParts)-1]
	log.Printf("repo path is %q", repoPath)
	//ssh := exec.Command("ssh")
	keyName := strings.TrimLeft(repoPath, "/")
	keyName = strings.ReplaceAll(keyName, "/", "_")

	var keyPath string

	{
		tryKeyPath := fmt.Sprintf("git_ssh_key_%s", keyName)
		_, err := os.Stat(tryKeyPath)
		if err == nil {
			log.Printf("got key override at path %s", tryKeyPath)

			keyPath, err = filepath.Abs(tryKeyPath)
			if err != nil {
				log.Printf("couldn't make %q absolute: %v", tryKeyPath, err)
				keyPath = tryKeyPath
			}
		} else {
			log.Printf("no key override at path %s", tryKeyPath)
		}
	}

	{
		envName := fmt.Sprintf("GIT_SSH_KEY_%s", strings.ToUpper(keyName))
		if keyStr := os.Getenv(envName); keyStr != "" {
			log.Printf("got key override from ENV %s", envName)
			tmpFile, err := os.CreateTemp(os.TempDir(), "repokey-*")
			if err != nil {
				panic(fmt.Errorf("error creating temp file: %+v", err))
			}
			_, err = tmpFile.WriteString(keyStr)
			if err != nil {
				panic(fmt.Errorf("error writing key to temp file: %+v", err))
			}
			err = tmpFile.Close()
			if err != nil {
				panic(fmt.Errorf("error closing temp file: %+v", err))
			}
			err = os.Chmod(tmpFile.Name(), 0600)
			if err != nil {
				panic(fmt.Errorf("error chmod'ing temp file: %+v", err))
			}
			defer func() {
				_ = os.Remove(tmpFile.Name())
			}()
			keyPath = tmpFile.Name()
		} else {
			log.Printf("no key override in ENV %s", envName)
		}
	}

	if keyPath != "" {
		sshParams = append([]string{"-i", keyPath}, sshParams...)
		log.Printf("new ssh params: %#v", sshParams)
	}

	sshCmd := exec.Command("ssh", sshParams...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr
	err = sshCmd.Run()
	if err != nil {
		log.Printf("error running ssh: %+v", err)
		if exitErr, is := err.(*exec.ExitError); is {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
}
