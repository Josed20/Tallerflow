package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
)

const bootstrapPassword = "Temporary secure passphrase 9!"

func TestRunBootstrapReadsPasswordFromStandardInputWithoutPrintingIt(t *testing.T) {
	creator := &bootstrapCreatorFake{result: auth.BootstrapResult{Role: "OWNER", MustChangePassword: true}}
	var stdout, stderr bytes.Buffer

	exitCode := runBootstrap(
		context.Background(),
		[]string{"--password-stdin", "--email", "owner@tallerflow.pe", "--name", "Owner Demo", "--workshop", "Taller Demo"},
		strings.NewReader(bootstrapPassword+"\n"),
		&stdout,
		&stderr,
		creator,
	)

	if exitCode != 0 {
		t.Fatalf("runBootstrap() exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if creator.input.Password != bootstrapPassword {
		t.Fatalf("bootstrap password = %q, want stdin value", creator.input.Password)
	}
	if strings.Contains(stdout.String(), bootstrapPassword) || strings.Contains(stderr.String(), bootstrapPassword) {
		t.Fatal("bootstrap command exposed the password in output")
	}
}

func TestReadPasswordNormalizesPipedLineEndings(t *testing.T) {
	for _, ending := range []string{"\n", "\r\n", "\r\r\n"} {
		password, err := readPassword(strings.NewReader(bootstrapPassword + ending))
		if err != nil {
			t.Fatalf("readPassword() with ending %q returned %v", ending, err)
		}
		if password != bootstrapPassword {
			t.Fatalf("readPassword() with ending %q retained line-ending bytes", ending)
		}
	}
	password, err := readPassword(strings.NewReader("\ufeff" + bootstrapPassword + "\r\n"))
	if err != nil {
		t.Fatalf("readPassword() with UTF-8 BOM returned %v", err)
	}
	if password != bootstrapPassword {
		t.Fatal("readPassword() retained the UTF-8 BOM")
	}
}

func TestRunBootstrapRejectsMissingPasswordStdinAndExistingOwner(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		err  error
		want string
	}{
		{
			name: "requires password stdin flag",
			args: []string{"--email", "owner@tallerflow.pe", "--name", "Owner Demo", "--workshop", "Taller Demo"},
			want: "BOOTSTRAP_INVALID_INPUT",
		},
		{
			name: "does not overwrite existing owner",
			args: []string{"--password-stdin", "--email", "owner@tallerflow.pe", "--name", "Owner Demo", "--workshop", "Taller Demo"},
			err:  auth.ErrBootstrapAlreadyExists,
			want: "BOOTSTRAP_ALREADY_EXISTS",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			creator := &bootstrapCreatorFake{err: tc.err}
			var stdout, stderr bytes.Buffer

			exitCode := runBootstrap(context.Background(), tc.args, strings.NewReader(bootstrapPassword+"\n"), &stdout, &stderr, creator)

			if exitCode == 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("runBootstrap() = %d, stderr = %q, want nonzero and %q", exitCode, stderr.String(), tc.want)
			}
			if strings.Contains(stdout.String(), bootstrapPassword) || strings.Contains(stderr.String(), bootstrapPassword) {
				t.Fatal("bootstrap command exposed the password in output")
			}
		})
	}
}

type bootstrapCreatorFake struct {
	input  auth.BootstrapInput
	result auth.BootstrapResult
	err    error
}

func (f *bootstrapCreatorFake) CreateOwner(_ context.Context, input auth.BootstrapInput) (auth.BootstrapResult, error) {
	f.input = input
	if f.err != nil {
		return auth.BootstrapResult{}, f.err
	}
	return f.result, nil
}
