package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var logger *log.Logger

func Exit1(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

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
	if a == nil {
		return b
	}
	var m Flags
	if a == nil && b == nil {
		return m
	}
	for k, v := range a {
		m[k] = v
	}
	for k, v := range b {
		m[k] = v
	}
	return m
}

func mergeEnv(a Env, b Env) Env {
	if b == nil && a != nil {
		return a
	}
	if a == nil && b != nil {
		return b
	}

	var m Env
	if a == nil && b == nil {
		return m
	}
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

	logger.Println("Command: ", bin)
	logger.Println("Command arguments: ", args)
	logger.Println("Command environment: ", env)

	toexec := exec.Command(bin, args...)
	toexec.Env = append(toexec.Env, env.ToArray()...)
	toexec.Env = append(toexec.Env, os.Environ()...)
	toexec.Stdout = os.Stdout
	toexec.Stdin = os.Stdin
	toexec.Stderr = os.Stderr

	if !dry {
		err := toexec.Run()
		if err != nil {
			Exit1(err)
		}
	}

}

var RootCmd = &cobra.Command{
	Use: "ecli",
	Run: func(cmd *cobra.Command, args []string) {
		if verbose, _ := cmd.PersistentFlags().GetBool("verbose"); verbose {
			logger.SetOutput(os.Stdout)
		} else {
			logger.SetOutput(ioutil.Discard)
		}
		configFile, _ := cmd.PersistentFlags().GetString("config")
		dry, _ := cmd.PersistentFlags().GetBool("dry")
		config, err := loadConfig(configFile)
		if err != nil {
			Exit1(fmt.Errorf("Error loading config %s: %s", configFile, err))
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
					Exit1(fmt.Errorf("Invalid Command: %s", command))
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
	logger = log.New(ioutil.Discard, "EASY_CLI - ", log.Ldate|log.Ltime)
}
