package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

type Args map[string]interface{}
type Env map[string]interface{}

type CommandConfig struct {
	Command string
	Args    map[string]interface{}
	Env     map[string]interface{}
}

func (c *CommandConfig) GetArgs() map[string]interface{} {
	return c.Args
}

func (c *CommandConfig) GetEnv() map[string]interface{} {
	return c.Env
}

type RunConfig struct {
	Bin      string
	Default  string
	Args     []string
	Flags    map[string]interface{}
	Env      map[string]interface{}
	Commands map[string]CommandConfig
}

func (r *RunConfig) GetArgs() Args {
	return r.Args
}

func (r *RunConfig) GetEnv() Env {
	return r.Env
}

type BaseConfig interface {
	GetArgs() Args
	GetEnv() Env
}

func mergeMaps(a map[string]interface{}, b map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func makeArgs(a map[string]interface{}) []string {

	var result []string
	for k, v := range a {
		switch v := v.(type) {
		default:
			result = append(result, k, fmt.Sprintf("%s", v))
		case float64:
			// TODO: nachkommastellen
			result = append(result, k, fmt.Sprintf("%.f", v))
		case bool:
			if v {
				result = append(result, k)
			}
		case []interface{}:
			for _, value := range v {
				result = append(result, k, fmt.Sprintf("%s", value))
			}
		}
	}
	return result
}

func makeEnv(input map[string]interface{}) []string {
	var result []string
	for k, v := range input {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func callCommand(bin string, c CommandConfig, defaults RunConfig) {

	args := makeArgs(mergeMaps(defaults.GetArgs(), c.GetArgs()))
	env := makeEnv(mergeMaps(defaults.GetEnv(), c.GetEnv()))

	if c.Command != "" {
		l := []string{c.Command}
		args = append(l, args...)
	}

	log.Println(args)
	log.Println(env)

	toexec := exec.Command(bin, args...)
	toexec.Env = append(toexec.Env, env...)
	toexec.Env = append(toexec.Env, os.Environ()...)
	toexec.Stdout = os.Stdout
	toexec.Stdin = os.Stdin
	toexec.Stderr = os.Stderr
	err := toexec.Run()
	if err != nil {
		fmt.Println("-------------------------------------------")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

var RootCmd = &cobra.Command{
	Use: "ecli",
	Run: func(cmd *cobra.Command, args []string) {
		config, _ := loadConfig()
		if len(args) > 0 || config.Default != "" {
			var commands []string
			if len(args) > 0 {
				commands = args
			} else {
				commands = []string{config.Default}
			}
			for _, command := range commands {
				if e, ok := config.Commands[command]; ok {
					callCommand(config.Bin, e, config)
				} else {
					fmt.Fprintf(os.Stderr, "Invalid Command: '%s'.\n", command)
					os.Exit(1)
				}
			}
		} else {
			callCommand(config.Bin, CommandConfig{}, config)
		}
	},
}

func loadConfig() (RunConfig, error) {
	var rc RunConfig
	f, _ := os.Open(".eclirc")
	defer f.Close()
	p := json.NewDecoder(f)
	err := p.Decode(&rc)
	if err != nil {
		panic(err)
	}

	return rc, nil
}

func init() {

}
