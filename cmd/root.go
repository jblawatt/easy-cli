package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type Flags map[string]interface{}
type Env map[string]string
type Args []string

func (e Env) ToArray() []string {
	var result []string
	for key, value := range e {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}
	return result
}

type CommandConfig struct {
	Args  Args  `json:"args"`
	Flags Flags `json:"flags"`
	Env   Env   `json:"env"`
}

func (c *CommandConfig) GetArgs() Args {
	return c.Args
}

func (c *CommandConfig) GetEnv() Env {
	return c.Env
}

func (c *CommandConfig) GetFlags() Flags {
	return c.Flags
}

type MainConfig struct {
	Bin      string                   `json:"bin"`
	Default  string                   `json:"default"`
	Flags    Flags                    `json:"flags"`
	Env      Env                      `json:"env"`
	Commands map[string]CommandConfig `json:"commands"`
}

func (r *MainConfig) GetEnv() Env {
	return r.Env
}

type Config interface {
	GetArgs() Args
	GetEnv() Env
	GetFlags() Flags
}

func mergeFlags(a Flags, b Flags) Flags {
	if b == nil {
		return a
	}
	var m Flags
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func mergeEnv(a Env, b Env) Env {
	if b == nil {
		return a
	}
	var m Env
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func (f Flags) FlagList() []string {
	var result []string
	for k, v := range f {
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

func transformValues(args []string) []string {
	var result []string
	envMap := make(map[string]string)
	for _, value := range os.Environ() {
		s := strings.Split(value, "=")
		envMap[s[0]] = s[1]
	}
	for _, arg := range args {
		t, err := template.New("").Parse(arg)
		if err != nil {
			result = append(result, arg)
		} else {
			var buff bytes.Buffer
			terr := t.Execute(&buff, envMap)
			if terr != nil {
				result = append(result, arg)
			} else {
				result = append(result, buff.String())
			}
		}

	}
	return result
}

func callCommand(bin string, c CommandConfig, defaults MainConfig, dry bool) {

	var args []string

	args = append(args, c.Args...)
	flags := mergeFlags(defaults.Flags, c.Flags)
	args = append(args, flags.FlagList()...)
	args = transformValues(args)

	env := mergeEnv(defaults.Env, c.Env)

	log.Println(args)
	log.Println(env)

	toexec := exec.Command(bin, args...)
	toexec.Env = append(toexec.Env, env.ToArray()...)
	toexec.Env = append(toexec.Env, os.Environ()...)
	toexec.Stdout = os.Stdout
	toexec.Stdin = os.Stdin
	toexec.Stderr = os.Stderr
	if !dry {
		err := toexec.Run()
		if err != nil {
			fmt.Println("-------------------------------------------")
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}

}

var RootCmd = &cobra.Command{
	Use: "ecli",
	Run: func(cmd *cobra.Command, args []string) {
		configFile, _ := cmd.PersistentFlags().GetString("config")
		dry, _ := cmd.PersistentFlags().GetBool("dry")
		config, err := loadConfig(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: '%s'.\n", configFile, err)
			os.Exit(1)
		}
		if len(args) > 0 || config.Default != "" {
			var commands []string
			if len(args) > 0 {
				commands = args
			} else {
				commands = []string{config.Default}
			}
			for _, command := range commands {
				if e, ok := config.Commands[command]; ok {
					callCommand(config.Bin, e, config, dry)
				} else {
					fmt.Fprintf(os.Stderr, "Invalid Command: '%s'.\n", command)
					os.Exit(1)
				}
			}
		} else {
			callCommand(config.Bin, CommandConfig{}, config, dry)
		}
	},
}

func loadConfig(configFile string) (MainConfig, error) {
	var rc MainConfig
	f, ferr := os.Open(configFile)
	if ferr != nil {
		return rc, ferr
	}
	defer f.Close()
	p := json.NewDecoder(f)
	err := p.Decode(&rc)
	if err != nil {
		return rc, err
	}
	return rc, nil
}

func init() {
	RootCmd.PersistentFlags().StringP("config", "c", ".eclirc", "easy-cli Config File. Default: .eclirc")
	RootCmd.PersistentFlags().BoolP("verbose", "v", false, "Tell me what you do.")
	RootCmd.PersistentFlags().BoolP("dry", "d", false, "Dry run.")
}
