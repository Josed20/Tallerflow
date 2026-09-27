package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxBootstrapPasswordBytes = 1 << 20

type bootstrapCreator interface {
	CreateOwner(context.Context, auth.BootstrapInput) (auth.BootstrapResult, error)
}

func main() {
	databaseURL := strings.TrimSpace(os.Getenv("TF_BOOTSTRAP_DATABASE_URL"))
	if databaseURL == "" {
		log.Print("bootstrap initialization failed: TF_BOOTSTRAP_DATABASE_URL is required")
		os.Exit(2)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Print("bootstrap initialization failed")
		os.Exit(2)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Print("bootstrap initialization failed")
		os.Exit(2)
	}

	service := auth.NewBootstrapService(
		auth.NewPostgresBootstrapStore(pool),
		auth.NewPasswordHasher(auth.DefaultPasswordParams()),
	)
	os.Exit(runBootstrap(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, service))
}

func runBootstrap(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, creator bootstrapCreator) int {
	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	flags.SetOutput(stderr)
	passwordStdin := flags.Bool("password-stdin", false, "read the initial password from standard input")
	email := flags.String("email", "", "owner email")
	name := flags.String("name", "", "owner name")
	workshop := flags.String("workshop", "", "workshop name")
	if err := flags.Parse(args); err != nil || !*passwordStdin || flags.NArg() != 0 {
		fmt.Fprintln(stderr, "BOOTSTRAP_INVALID_INPUT")
		return 2
	}

	password, err := readPassword(stdin)
	if err != nil {
		fmt.Fprintln(stderr, "BOOTSTRAP_INVALID_INPUT")
		return 2
	}
	_, err = creator.CreateOwner(ctx, auth.BootstrapInput{
		Email: *email, Name: *name, WorkshopName: *workshop, Password: password,
	})
	switch {
	case err == nil:
		fmt.Fprintln(stdout, "OWNER_BOOTSTRAPPED")
		return 0
	case errors.Is(err, auth.ErrBootstrapAlreadyExists):
		fmt.Fprintln(stderr, "BOOTSTRAP_ALREADY_EXISTS")
		return 1
	case errors.Is(err, auth.ErrBootstrapInvalidInput):
		fmt.Fprintln(stderr, "BOOTSTRAP_INVALID_INPUT")
		return 2
	default:
		fmt.Fprintln(stderr, "BOOTSTRAP_FAILED")
		return 1
	}
}

func readPassword(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", errors.New("password input is required")
	}
	contents, err := io.ReadAll(io.LimitReader(stdin, maxBootstrapPasswordBytes+9))
	if err != nil || len(contents) == 0 || len(contents) > maxBootstrapPasswordBytes+8 {
		return "", errors.New("invalid password input")
	}
	password := strings.TrimPrefix(strings.TrimRight(string(contents), "\r\n"), "\ufeff")
	if password == "" || len(password) > maxBootstrapPasswordBytes {
		return "", errors.New("invalid password input")
	}
	return password, nil
}
