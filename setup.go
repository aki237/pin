package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"gitlab.com/aki237/pin/pinlib"
)

func executeTemplate(script string, env interface{}) (string, error) {
	sb := bytes.NewBuffer(nil)
	t, err := template.New("script").Parse(script)
	if err != nil {
		return "", err
	}

	err = t.Execute(sb, env)
	return sb.String(), err
}

func runScript(script string) error {
	fmt.Printf("Executing:\n\t%s\n", strings.ReplaceAll(script, "\n", "\n\t"))
	cmd := exec.Command("sh", "-c", script)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (s *Session) SetupClient() {
	client, ok := s.peer.(*pinlib.Client)
	if !ok {
		return
	}

	client.Hook = func(ipp, gw string) error {

		s.InterfaceAddress = ipp
		s.InterfaceGateway = gw

		scriptTmpl, ok := s.Config.PostConnectScript[runtime.GOOS]
		if !ok {
			fmt.Printf("[WARN] No post connect script defined for '%s' platform", runtime.GOOS)
			return nil
		}

		script, err := executeTemplate(scriptTmpl, map[string]interface{}{
			"interfaceName": s.InterfaceName,
			"mtu":           s.MTU,
			"remoteIP":      s.ResolvedRemoteIP.String(),
			"remotePort":    s.RemotePort,
			"tunIP":         ipp,
			"tunGateway":    gw,
			"dns":           s.DNS,
		})
		if err != nil {
			return err
		}

		return runScript(script)
	}
}

func (s *Session) SetupServer() error {
	scriptTmpl, ok := s.Config.PostInitScript[runtime.GOOS]
	if !ok {
		fmt.Printf("[WARN] No post init script defined for '%s' platform", runtime.GOOS)
		return nil
	}

	script, err := executeTemplate(scriptTmpl, map[string]interface{}{
		"interfaceName": s.InterfaceName,
		"mtu":           s.MTU,
		"tunIP":         s.DHCP,
		"dns":           s.DNS,
	})
	if err != nil {
		return err
	}

	return runScript(script)
}

func (s *Session) StopClient() error {
	scriptTmpl, ok := s.Config.PostDisconnectScript[runtime.GOOS]
	if !ok {
		fmt.Printf("[WARN] No post init script defined for '%s' platform", runtime.GOOS)
		return nil
	}

	script, err := executeTemplate(scriptTmpl, map[string]interface{}{
		"interfaceName": s.InterfaceName,
		"remoteIP":      s.ResolvedRemoteIP.String(),
		"remotePort":    s.RemotePort,
		"mtu":           s.MTU,
		"tunIP":         s.DHCP,
		"dns":           s.DNS,
	})
	if err != nil {
		return err
	}

	return runScript(script)
}
