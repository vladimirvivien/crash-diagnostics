// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package starlark

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"

	"github.com/vmware-tanzu/crash-diagnostics/ssh"
)

func testCopyFromFuncForHostResources(t *testing.T, port, privateKey, username string) {
	tests := []struct {
		name        string
		remoteFiles map[string]string
		args        func(t *testing.T) starlark.Tuple
		kwargs      func(t *testing.T) []starlark.Tuple
		eval        func(t *testing.T, args starlark.Tuple, kwargs []starlark.Tuple)
	}{
		{
			name:        "single machine single file",
			remoteFiles: map[string]string{"foo.txt": "FooBar"},
			args:        func(t *testing.T) starlark.Tuple { return starlark.Tuple{starlark.String("foo.txt")} },
			kwargs: func(t *testing.T) []starlark.Tuple {
				sshCfg := makeTestSSHConfig(privateKey, port, username)
				resources := starlark.NewList([]starlark.Value{makeTestSSHHostResource("127.0.0.1", sshCfg)})
				return []starlark.Tuple{[]starlark.Value{starlark.String("resources"), resources}}
			},

			eval: func(t *testing.T, args starlark.Tuple, kwargs []starlark.Tuple) {

				val, err := copyFromFunc(newTestThreadLocal(t), nil, args, kwargs)
				if err != nil {
					t.Fatal(err)
				}
				resource := ""
				cpErr := ""
				result := ""
				if strct, ok := val.(*starlarkstruct.Struct); ok {
					if val, err := strct.Attr("resource"); err == nil {
						if r, ok := val.(starlark.String); ok {
							resource = string(r)
						}
					}
					if val, err := strct.Attr("err"); err == nil {
						if r, ok := val.(starlark.String); ok {
							cpErr = string(r)
						}
					}
					if val, err := strct.Attr("result"); err == nil {
						if r, ok := val.(starlark.String); ok {
							result = string(r)
						}
					}
				}

				if cpErr != "" {
					t.Fatal(cpErr)
				}

				expected := filepath.Join(defaults.workdir, sanitizeStr(resource), "foo.txt")
				if result != expected {
					t.Errorf("unexpected file name copied: %s", result)
				}

				defer os.RemoveAll(expected)
			},
		},
		{
			name:        "single machine compress",
			remoteFiles: map[string]string{"foo.txt": "FooBar"},
			args:        func(t *testing.T) starlark.Tuple { return starlark.Tuple{starlark.String("foo.txt")} },
			kwargs: func(t *testing.T) []starlark.Tuple {
				sshCfg := makeTestSSHConfig(privateKey, port, username)
				resources := starlark.NewList([]starlark.Value{makeTestSSHHostResource("127.0.0.1", sshCfg)})
				return []starlark.Tuple{
					[]starlark.Value{starlark.String("resources"), resources},
					[]starlark.Value{starlark.String("compress"), starlark.Bool(true)},
				}
			},

			eval: func(t *testing.T, args starlark.Tuple, kwargs []starlark.Tuple) {

				val, err := copyFromFunc(newTestThreadLocal(t), nil, args, kwargs)
				if err != nil {
					t.Fatal(err)
				}
				resource := ""
				cpErr := ""
				result := ""
				if strct, ok := val.(*starlarkstruct.Struct); ok {
					if val, err := strct.Attr("resource"); err == nil {
						if r, ok := val.(starlark.String); ok {
							resource = string(r)
						}
					}
					if val, err := strct.Attr("err"); err == nil {
						if r, ok := val.(starlark.String); ok {
							cpErr = string(r)
						}
					}
					if val, err := strct.Attr("result"); err == nil {
						if r, ok := val.(starlark.String); ok {
							result = string(r)
						}
					}
				}

				if cpErr != "" {
					t.Fatal(cpErr)
				}

				expected := filepath.Join(defaults.workdir, sanitizeStr(resource), "foo.txt")
				if result != expected {
					t.Errorf("unexpected file name copied: %s", result)
				}

				defer os.RemoveAll(expected)
			},
		},
		{
			name:        "multiple machines single files",
			remoteFiles: map[string]string{"bar/bar.txt": "BarBar", "bar/foo.txt": "FooBar", "baz.txt": "BazBuz"},
			args:        func(t *testing.T) starlark.Tuple { return nil },
			kwargs: func(t *testing.T) []starlark.Tuple {
				sshCfg := makeTestSSHConfig(privateKey, port, username)
				resources := starlark.NewList([]starlark.Value{
					makeTestSSHHostResource("localhost", sshCfg),
					makeTestSSHHostResource("127.0.0.1", sshCfg),
				})
				return []starlark.Tuple{
					[]starlark.Value{starlark.String("path"), starlark.String("bar/bar.txt")},
					[]starlark.Value{starlark.String("resources"), resources},
				}
			},
			eval: func(t *testing.T, args starlark.Tuple, kwargs []starlark.Tuple) {
				val, err := copyFromFunc(newTestThreadLocal(t), nil, args, kwargs)
				if err != nil {
					t.Fatal(err)
				}

				resultList, ok := val.(*starlark.List)
				if !ok {
					t.Fatalf("expecting type *starlark.List, got %T", val)
				}

				for i := 0; i < resultList.Len(); i++ {
					resource := ""
					cpErr := ""
					result := ""
					if strct, ok := resultList.Index(i).(*starlarkstruct.Struct); ok {
						if val, err := strct.Attr("resource"); err == nil {
							if r, ok := val.(starlark.String); ok {
								resource = string(r)
							}
						}
						if val, err := strct.Attr("err"); err == nil {
							if r, ok := val.(starlark.String); ok {
								cpErr = string(r)
							}
						}
						if val, err := strct.Attr("result"); err == nil {
							if r, ok := val.(starlark.String); ok {
								result = string(r)
							}
						}
					}

					if cpErr != "" {
						t.Fatal(cpErr)
					}

					expected := filepath.Join(defaults.workdir, sanitizeStr(resource), "bar/bar.txt")
					if result != expected {
						t.Errorf("expecting copied file %s, got %s", expected, result)
					}
					os.RemoveAll(result)
				}
			},
		},

		{
			name:        "multiple machines files glob",
			remoteFiles: map[string]string{"bar/bar.txt": "BarBar", "bar/foo.txt": "FooBar", "bar/baz.csv": "BizzBuzz"},
			args:        func(t *testing.T) starlark.Tuple { return nil },
			kwargs: func(t *testing.T) []starlark.Tuple {
				sshCfg := makeTestSSHConfig(privateKey, port, username)
				resources := starlark.NewList([]starlark.Value{
					makeTestSSHHostResource("localhost", sshCfg),
					makeTestSSHHostResource("127.0.0.1", sshCfg),
				})
				return []starlark.Tuple{
					[]starlark.Value{starlark.String("path"), starlark.String("bar/*.txt")},
					[]starlark.Value{starlark.String("resources"), resources},
				}
			},
			eval: func(t *testing.T, args starlark.Tuple, kwargs []starlark.Tuple) {
				val, err := copyFromFunc(newTestThreadLocal(t), nil, args, kwargs)
				if err != nil {
					t.Fatal(err)
				}

				resultList, ok := val.(*starlark.List)
				if !ok {
					t.Fatalf("expecting type *starlark.List, got %T", val)
				}

				for i := 0; i < resultList.Len(); i++ {
					resource := ""
					cpErr := ""
					result := ""
					if strct, ok := resultList.Index(i).(*starlarkstruct.Struct); ok {
						if val, err := strct.Attr("resource"); err == nil {
							if r, ok := val.(starlark.String); ok {
								resource = string(r)
							}
						}
						if val, err := strct.Attr("err"); err == nil {
							if r, ok := val.(starlark.String); ok {
								cpErr = string(r)
							}
						}
						if val, err := strct.Attr("result"); err == nil {
							if r, ok := val.(starlark.String); ok {
								result = string(r)
							}
						}
					}

					if cpErr != "" {
						t.Fatal(cpErr)
					}

					path := filepath.Join(defaults.workdir, sanitizeStr(resource), "bar")
					finfos, err := ioutil.ReadDir(path)
					if err != nil {
						t.Fatal(err)
					}
					if len(finfos) != 2 {
						t.Errorf("expecting 2 files copied, got %d", len(finfos))
					}

					os.RemoveAll(result)
				}
			},
		},
	}

	sshArgs := ssh.SSHArgs{
		User:           username,
		Host:           "127.0.0.1",
		Port:           port,
		PrivateKeyPath: privateKey,
		MaxRetries:     testSupport.MaxConnectionRetries(),
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for file, content := range test.remoteFiles {
				ssh.MakeRemoteTestSSHFile(t, sshArgs, file, content)
			}
			defer func() {
				for file := range test.remoteFiles {
					ssh.RemoveRemoteTestSSHFile(t, sshArgs, file)
				}
			}()

			test.eval(t, test.args(t), test.kwargs(t))
		})
	}
}

func testCopyFromFuncScriptForHostResources(t *testing.T, port, privateKey, username string) {
	tests := []struct {
		name        string
		remoteFiles map[string]string
		script      string
		eval        func(t *testing.T, script string)
	}{
		{
			name:        "multiple machines single copyFrom",
			remoteFiles: map[string]string{"foobar.c": "footext", "bar/bar.txt": "BarBar", "bar/foo.txt": "FooBar", "bar/baz.csv": "BizzBuzz"},
			script: fmt.Sprintf(`
set_defaults(resources(provider = host_list_provider(hosts=["127.0.0.1","localhost"], ssh_config = ssh_config(username="%s", port="%s", private_key_path="%s"))))
result = copy_from("bar/foo.txt")`, username, port, privateKey),
			eval: func(t *testing.T, script string) {
				exe := New()
				if err := exe.Exec("test.star", strings.NewReader(script)); err != nil {
					t.Fatal(err)
				}

				resultVal := exe.result["result"]
				if resultVal == nil {
					t.Fatal("capture() should be assigned to a variable")
				}
				resultList, ok := resultVal.(*starlark.List)
				if !ok {
					t.Fatalf("expecting type *starlark.List, got %T", resultVal)
				}

				for i := 0; i < resultList.Len(); i++ {
					resource := ""
					cpErr := ""
					result := ""
					if strct, ok := resultList.Index(i).(*starlarkstruct.Struct); ok {
						if val, err := strct.Attr("resource"); err == nil {
							if r, ok := val.(starlark.String); ok {
								resource = string(r)
							}
						}
						if val, err := strct.Attr("err"); err == nil {
							if r, ok := val.(starlark.String); ok {
								cpErr = string(r)
							}
						}
						if val, err := strct.Attr("result"); err == nil {
							if r, ok := val.(starlark.String); ok {
								result = string(r)
							}
						}
					}

					if cpErr != "" {
						t.Fatal(cpErr)
					}

					path := filepath.Join(defaults.workdir, sanitizeStr(resource), "bar/foo.txt")
					if result != path {
						t.Errorf("unexpected %s, got %s", path, result)
					}

					os.RemoveAll(result)
				}
			},
		},

		{
			name: "resource loop",
			script: fmt.Sprintf(`
# execute cmd on each host
def cp(hosts):
	result = []
	for host in hosts:
		result.append(copy_from(path="bar/*.txt", resources=[host]))
		return result

# configuration
set_defaults(ssh_config(username="%s", port="%s", private_key_path="%s"))
hosts = resources(provider=host_list_provider(hosts=["127.0.0.1","localhost"]))
result = cp(hosts)`, username, port, privateKey),
			eval: func(t *testing.T, script string) {
				exe := New()
				if err := exe.Exec("test.star", strings.NewReader(script)); err != nil {
					t.Fatal(err)
				}

				resultVal := exe.result["result"]
				if resultVal == nil {
					t.Fatal("capture() should be assigned to a variable")
				}
				resultList, ok := resultVal.(*starlark.List)
				if !ok {
					t.Fatalf("expecting type *starlark.List, got %T", resultVal)
				}

				for i := 0; i < resultList.Len(); i++ {
					resource := ""
					cpErr := ""
					result := ""
					if strct, ok := resultList.Index(i).(*starlarkstruct.Struct); ok {
						if val, err := strct.Attr("resource"); err == nil {
							if r, ok := val.(starlark.String); ok {
								resource = string(r)
							}
						}
						if val, err := strct.Attr("err"); err == nil {
							if r, ok := val.(starlark.String); ok {
								cpErr = string(r)
							}
						}
						if val, err := strct.Attr("result"); err == nil {
							if r, ok := val.(starlark.String); ok {
								result = string(r)
							}
						}
					}

					if cpErr != "" {
						t.Fatal(cpErr)
					}

					path := filepath.Join(defaults.workdir, sanitizeStr(resource), "bar")
					finfos, err := ioutil.ReadDir(path)
					if err != nil {
						t.Fatal(err)
					}
					if len(finfos) != 2 {
						t.Errorf("expecting 2 files copied, got %d", len(finfos))
					}

					os.RemoveAll(result)
				}
			},
		},
	}

	sshArgs := ssh.SSHArgs{User: username, Host: "127.0.0.1", Port: port, PrivateKeyPath: privateKey}
	for _, test := range tests {
		for file, content := range test.remoteFiles {
			ssh.MakeRemoteTestSSHFile(t, sshArgs, file, content)
		}
		defer func() {
			for file := range test.remoteFiles {
				ssh.RemoveRemoteTestSSHFile(t, sshArgs, file)
			}
		}()

		t.Run(test.name, func(t *testing.T) {
			test.eval(t, test.script)
		})
	}
}

func TestCopyFromFuncSSHAll(t *testing.T) {
	port := testSupport.PortValue()
	username := testSupport.CurrentUsername()
	privateKey := testSupport.PrivateKeyPath()

	tests := []struct {
		name string
		test func(t *testing.T, port, privateKey, username string)
	}{
		{name: "copyFrom func for host resources", test: testCopyFromFuncForHostResources},
		{name: "copy_from script for host resources", test: testCopyFromFuncScriptForHostResources},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.test(t, port, privateKey, username)
			defer os.RemoveAll(defaults.workdir)
		})
	}
}
