package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/cloudwego/thriftgo/plugin"

	generator "github.com/sunyakun/thriftgo-tools"
)

func PluginMode() {
	var req *plugin.Request
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}
	req, err = plugin.UnmarshalRequest(data)
	if err != nil {
		panic(err)
	}

	g := generator.NewGenerator()
	resp, err := g.Execute(req)
	if err != nil {
		panic(err)
	}

	rb, err := plugin.MarshalResponse(resp)
	if err != nil {
		panic(err)
	}
	_, _ = os.Stdout.Write(rb)
}

func ProgramMode(pluginPath string) {
	var (
		outputPath    string
		module        string
		routerPath    string
		handlerPath   string
		servicePath   string
		packagePrefix string
		thriftFile    string
	)

	flag.StringVar(&outputPath, "output", "", "output path")
	flag.StringVar(&module, "module", "", "module name")
	flag.StringVar(&routerPath, "router", "", "router file path")
	flag.StringVar(&handlerPath, "handler", "", "handler file path")
	flag.StringVar(&servicePath, "service", "", "service file path")
	flag.StringVar(&packagePrefix, "prefix", "", "package prefix")
	thriftFile = os.Args[len(os.Args)-1]
	flag.Parse()

	empty := func(val, name string) error {
		if val == "" {
			return fmt.Errorf("%s is empty", name)
		}
		return nil
	}

	if err := empty(outputPath, "output"); err != nil {
		panic(err)
	}

	thriftgoArgs := []string{"-o", outputPath, "-g"}
	if packagePrefix != "" {
		thriftgoArgs = append(thriftgoArgs, "go:package_prefix="+packagePrefix)
	} else {
		thriftgoArgs = append(thriftgoArgs, "go")
	}

	pluginArgs := []string{}
	if routerPath != "" {
		pluginArgs = append(pluginArgs, "router="+routerPath)
	}
	if handlerPath != "" {
		pluginArgs = append(pluginArgs, "handler="+handlerPath)
	}
	if servicePath != "" {
		pluginArgs = append(pluginArgs, "service="+servicePath)
	}
	if module != "" {
		pluginArgs = append(pluginArgs, "module="+module)
	}
	thriftgoArgs = append(thriftgoArgs, "--plugin", "plugin="+pluginPath+":"+strings.Join(pluginArgs, ","))
	thriftgoArgs = append(thriftgoArgs, thriftFile)

	cmd := exec.Command("thriftgo", thriftgoArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok {
			panic(err)
		} else {
			os.Exit(exitErr.ExitCode())
		}
	}
}

func main() {
	executable, err := os.Executable()
	if err != nil {
		panic(err)
	}

	if strings.HasSuffix(path.Base(executable), "plugin") {
		PluginMode()
		return
	}

	executableDir := path.Dir(executable)
	pluginPath := path.Join(executableDir, "thriftgo-http-plugin")
	if fileInfo, err := os.Stat(pluginPath); err != nil || fileInfo.IsDir() {
		if err != nil {
			pluginProgram, err := os.OpenFile(pluginPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				panic(err)
			}
			currentProgram, err := os.Open(executable)
			if err != nil {
				panic(err)
			}
			if _, err := io.Copy(pluginProgram, currentProgram); err != nil {
				panic(err)
			}
		} else if fileInfo.IsDir() {
			panic(fmt.Errorf("%s is a directory", pluginPath))
		}
	}
	ProgramMode(pluginPath)
}
