package cmd

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"github.com/meain/refer/internal"
)

type Add struct {
	FilePath []string
	NoIgnore bool
}

type Search struct {
	Query     []string
	Format    string
	Limit     int
	Threshold *float64
	Rerank    bool
}

type Reindex struct{}

type Watch struct {
	Path string
}

type Show struct {
	ID *int
}

type StatsCmd struct{}

type Remove struct {
	ID int
}

type App struct {
	Context      context.Context
	Database     *sql.DB
	Config       *internal.Config
	DatabasePath string
}

type runner interface {
	Run(*App) error
}

type commandSpec struct {
	description string
	build       func() runner
	bindFlags   func(*flag.FlagSet, runner)
	validate    func(*flag.FlagSet, runner) error
}

type commandFactory func() commandSpec

var commandFactories = map[string]commandFactory{
	"add": func() commandSpec {
		return commandSpec{
			description: "Add a file or directory to the database",
			build: func() runner {
				return &Add{}
			},
			bindFlags: func(fs *flag.FlagSet, r runner) {
				cmd := r.(*Add)
				fs.BoolVar(&cmd.NoIgnore, "no-ignore", false, "Do not ignore files that are ignored by git")
			},
			validate: func(fs *flag.FlagSet, r runner) error {
				cmd := r.(*Add)
				if len(fs.Args()) == 0 {
					return fmt.Errorf("add requires at least one file, directory or URL")
				}
				cmd.FilePath = fs.Args()
				return nil
			},
		}
	},
	"search": func() commandSpec {
		threshold := optionalFloat64{}
		return commandSpec{
			description: "Search for documents",
			build: func() runner {
				return &Search{}
			},
			bindFlags: func(fs *flag.FlagSet, r runner) {
				cmd := r.(*Search)
				fs.StringVar(&cmd.Format, "format", "names", "Format of the search results")
				fs.IntVar(&cmd.Limit, "limit", 5, "Maximum number of search results to return")
				fs.Var(&threshold, "threshold", "Maximum distance threshold for search results (20 is a good value)")
				fs.BoolVar(&cmd.Rerank, "rerank", false, "Rerank search results based on the query (alpha)")
			},
			validate: func(fs *flag.FlagSet, r runner) error {
				cmd := r.(*Search)
				cmd.Query = fs.Args()
				cmd.Threshold = threshold.ptr()
				return nil
			},
		}
	},
	"show": func() commandSpec {
		return commandSpec{
			description: "List documents in the database",
			build: func() runner {
				return &Show{}
			},
			validate: func(fs *flag.FlagSet, r runner) error {
				cmd := r.(*Show)
				if len(fs.Args()) > 1 {
					return fmt.Errorf("show accepts at most one document ID")
				}
				if len(fs.Args()) == 0 {
					return nil
				}

				var id int
				if _, err := fmt.Sscanf(fs.Args()[0], "%d", &id); err != nil {
					return fmt.Errorf("invalid document ID: %s", fs.Args()[0])
				}
				cmd.ID = &id
				return nil
			},
		}
	},
	"stats": func() commandSpec {
		return commandSpec{
			description: "Show database statistics",
			build: func() runner {
				return &StatsCmd{}
			},
			validate: func(fs *flag.FlagSet, _ runner) error {
				if len(fs.Args()) != 0 {
					return fmt.Errorf("stats does not accept positional arguments")
				}
				return nil
			},
		}
	},
	"reindex": func() commandSpec {
		return commandSpec{
			description: "Reindex all documents",
			build: func() runner {
				return &Reindex{}
			},
			validate: func(fs *flag.FlagSet, _ runner) error {
				if len(fs.Args()) != 0 {
					return fmt.Errorf("reindex does not accept positional arguments")
				}
				return nil
			},
		}
	},
	"remove": func() commandSpec {
		return commandSpec{
			description: "Remove a document from the database",
			build: func() runner {
				return &Remove{}
			},
			validate: func(fs *flag.FlagSet, r runner) error {
				cmd := r.(*Remove)
				if len(fs.Args()) != 1 {
					return fmt.Errorf("remove requires a document ID")
				}

				if _, err := fmt.Sscanf(fs.Args()[0], "%d", &cmd.ID); err != nil {
					return fmt.Errorf("invalid document ID: %s", fs.Args()[0])
				}
				return nil
			},
		}
	},
	"watch": func() commandSpec {
		return commandSpec{
			description: "Watch a directory and index files automatically",
			build: func() runner {
				return &Watch{Path: "."}
			},
			validate: func(fs *flag.FlagSet, r runner) error {
				cmd := r.(*Watch)
				if len(fs.Args()) > 1 {
					return fmt.Errorf("watch accepts at most one path")
				}
				if len(fs.Args()) == 1 {
					cmd.Path = fs.Args()[0]
				}
				return nil
			},
		}
	},
}

func Execute() {
	ctx := context.Background()
	cfg := loadConfig()

	databasePath, commandName, command, err := parseCLI(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatal(err)
	}

	database, isNew, err := setupDatabase(ctx, databasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	if err := validateEmbeddingModel(database, isNew, commandName, cfg); err != nil {
		log.Fatal(err)
	}

	app := &App{
		Context:      ctx,
		Database:     database,
		Config:       cfg,
		DatabasePath: databasePath,
	}

	if err := command.Run(app); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() *internal.Config {
	cfg, err := internal.LoadConfig()
	if err != nil {
		log.Printf("Warning: using default config: %v", err)
	}

	return cfg
}

func setupDatabase(ctx context.Context, path string) (*sql.DB, bool, error) {
	database, isNew, err := internal.CreateDB(path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create database: %w", err)
	}

	if !isNew {
		return database, false, nil
	}

	sampleEmbedding, err := internal.CreateEmbedding(ctx, "refer")
	if err != nil {
		return nil, false, fmt.Errorf("failed to create embedding: %w", err)
	}

	if err := internal.InitDatabase(database, len(sampleEmbedding)); err != nil {
		return nil, false, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err := internal.SaveConfig(
		database,
		map[string]string{
			"embedding_model": internal.Model,
			"embedding_size":  fmt.Sprintf("%d", len(sampleEmbedding)),
		},
	); err != nil {
		return nil, false, fmt.Errorf("failed to save config: %w", err)
	}

	return database, true, nil
}

func validateEmbeddingModel(database *sql.DB, isNew bool, command string, cfg *internal.Config) error {
	if isNew {
		return nil
	}

	if !requiresEmbeddingModelValidation(command) {
		return nil
	}

	config, err := internal.GetConfig(database)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if config["embedding_model"] == cfg.EmbeddingModel {
		return nil
	}

	return fmt.Errorf(
		"Database embedding model does not match config: %s != %s\n"+
			"Please reindex the documents or update the model\n",
		config["embedding_model"],
		cfg.EmbeddingModel,
	)
}

func readAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

func requiresEmbeddingModelValidation(command string) bool {
	return command == "add" || command == "search" || command == "watch"
}

func parseCLI(args []string) (string, string, runner, error) {
	root := flag.NewFlagSet("refer", flag.ContinueOnError)
	root.SetOutput(io.Discard)

	database := root.String("database", ".refer/refer.db", "Database file path")
	root.Usage = func() {
		printRootUsage()
	}

	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			root.Usage()
		}
		return "", "", nil, err
	}

	rest := root.Args()
	if len(rest) == 0 {
		root.Usage()
		return "", "", nil, flag.ErrHelp
	}

	commandName := rest[0]
	commandArgs := rest[1:]

	command, err := parseCommand(commandName, commandArgs)
	if err != nil {
		return "", "", nil, err
	}

	return *database, commandName, command, nil
}

func parseCommand(name string, args []string) (runner, error) {
	if name == "help" {
		printRootUsage()
		return nil, flag.ErrHelp
	}

	factory, ok := commandFactories[name]
	if !ok {
		return nil, fmt.Errorf("unknown command: %s", name)
	}

	spec := factory()
	fs := newFlagSet(name, spec.description)
	cmd := spec.build()
	if spec.bindFlags != nil {
		spec.bindFlags(fs, cmd)
	}
	if err := parseFlagSet(fs, args); err != nil {
		return nil, err
	}
	if spec.validate == nil {
		return cmd, nil
	}
	if err := spec.validate(fs, cmd); err != nil {
		return nil, err
	}
	return cmd, nil
}

func newFlagSet(name, description string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		printCommandUsage(name, description)
	}
	return fs
}

func parseFlagSet(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.Usage()
		}
		return err
	}
	return nil
}

func printRootUsage() {
	fmt.Fprintf(os.Stderr, "Usage: refer [--database PATH] <command> [options]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  add      Add a file or directory to the database\n")
	fmt.Fprintf(os.Stderr, "  search   Search for documents\n")
	fmt.Fprintf(os.Stderr, "  show     List documents in the database\n")
	fmt.Fprintf(os.Stderr, "  stats    Show database statistics\n")
	fmt.Fprintf(os.Stderr, "  reindex  Reindex all documents\n")
	fmt.Fprintf(os.Stderr, "  remove   Remove a document from the database\n")
	fmt.Fprintf(os.Stderr, "  watch    Watch a directory and index files automatically\n")
}

func printCommandUsage(name, description string) {
	fmt.Fprintf(os.Stderr, "%s\n\n", description)
	fmt.Fprintf(os.Stderr, "Usage: refer %s [options]\n", name)
}

type optionalFloat64 struct {
	value float64
	set   bool
}

func (o *optionalFloat64) String() string {
	if !o.set {
		return ""
	}
	return fmt.Sprintf("%v", o.value)
}

func (o *optionalFloat64) Set(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	o.value = parsed
	o.set = true
	return nil
}

func (o *optionalFloat64) ptr() *float64 {
	if !o.set {
		return nil
	}
	return &o.value
}
