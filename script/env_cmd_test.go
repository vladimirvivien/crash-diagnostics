package script

import (
	"fmt"
	"testing"
)

func TestCommandENV(t *testing.T) {
	tests := []commandTest{
		{
			name: "ENV with single arg",
			source: func() string {
				return "ENV foo=bar"
			},
			script: func(s *Script) error {
				envs := s.Preambles[CmdEnv]
				if len(envs) != 1 {
					return fmt.Errorf("Script has unexpected number of ENV %d", len(envs))
				}
				envCmd, ok := envs[0].(*EnvCommand)
				if !ok {
					return fmt.Errorf("Unexpected type %T in script", envs[0])
				}
				if len(envCmd.Envs()) != 1 {
					return fmt.Errorf("ENV has unexpected number of env %d", len(envCmd.Envs()))
				}
				env := envCmd.Envs()[0]
				if env != "foo=bar" {
					return fmt.Errorf("ENV has unexpected env %s", env)
				}
				return nil
			},
		},
		{
			name: "Multiple ENV with multiple args",
			source: func() string {
				return "ENV a=b\nENV c=d e=f"
			},
			script: func(s *Script) error {
				envs := s.Preambles[CmdEnv]
				if len(envs) != 2 {
					return fmt.Errorf("Script has unexpected number of ENV %d", len(envs))
				}

				envCmd0, ok := envs[0].(*EnvCommand)
				if !ok {
					return fmt.Errorf("Unexpected type %T in script", envs[0])
				}
				if len(envCmd0.Envs()) != 1 {
					return fmt.Errorf("ENV[0] has unexpected number of env %d", len(envCmd0.Envs()))
				}
				env := envCmd0.Envs()[0]
				if env != "a=b" {
					return fmt.Errorf("ENV[0] has unexpected env %s", env)
				}

				envCmd1, ok := envs[1].(*EnvCommand)
				if !ok {
					return fmt.Errorf("Unexpected type %T in script", envs[1])
				}

				if len(envCmd1.Envs()) != 2 {
					return fmt.Errorf("ENV[1] has unexpected number of env %d", len(envCmd1.Envs()))
				}
				env0, env1 := envCmd1.Envs()[0], envCmd1.Envs()[1]
				if env0 != "c=d" || env1 != "e=f" {
					return fmt.Errorf("ENV[1] has unexpected env[0] %s and env[1] %s", env0, env1)
				}
				return nil
			},
		},
		{
			name: "ENV with bad formatted values",
			source: func() string {
				return "ENV a=b foo|bar"
			},
			shouldFail: true,
		},
		{
			name: "ENV with missing env",
			source: func() string {
				return "ENV "
			},
			shouldFail: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runCommandTest(t, test)
		})
	}
}