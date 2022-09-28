package cmd

import (
	"log"
	"os"
	"strings"

	"github.com/redhat-et/copilot-ops/pkg/ai"
	"github.com/redhat-et/copilot-ops/pkg/cmd/config"
	"github.com/redhat-et/copilot-ops/pkg/filemap"
	"github.com/spf13/cobra"
)

// Request Defines the necessary values used when requesting new files from the selected
// AI backends.
// FIXME: consolidate the settings depending on the type of Model. E.g., OpenAI settings should be under their own.
type Request struct {
	Config      config.Config
	Fileset     *config.Filesets
	Filemap     *filemap.Filemap
	FilemapText string
	UserRequest string
	IsWrite     bool
	OutputType  string
}

// Flags Defines all of the values extracted from the commandline.
type Flags struct {
	Request      string
	Write        bool
	Path         string
	Files        []string
	Filesets     []string
	NTokens      int32
	NCompletions int32
	OutputType   string
	AIBackend    string
}

// ExtractFlags Returns a struct containing all of the flags present in
// in the provided command.
func ExtractFlags(cmd *cobra.Command) Flags {
	request, _ := cmd.Flags().GetString(FlagRequestFull)
	write, _ := cmd.Flags().GetBool(FlagWriteFull)
	path, _ := cmd.Flags().GetString(FlagPathFull)
	files, _ := cmd.Flags().GetStringArray(FlagFilesFull)
	if cmd.Name() == CommandEdit {
		file, _ := cmd.Flags().GetString(FlagFilesFull)
		files = append(files, file)
	}
	filesets, _ := cmd.Flags().GetStringArray(FlagFilesetsFull)
	nTokens, _ := cmd.Flags().GetInt32(FlagNTokensFull)
	nCompletions, _ := cmd.Flags().GetInt32(FlagNCompletionsFull)
	outputType, _ := cmd.Flags().GetString(FlagOutputTypeFull)
	aiBackend, _ := cmd.Flags().GetString(FlagAIBackendFull)

	log.Println("flags:")
	log.Printf(" - %-8s: %v\n", FlagRequestFull, request)
	log.Printf(" - %-8s: %v\n", FlagWriteFull, write)
	log.Printf(" - %-8s: %v\n", FlagPathFull, path)
	log.Printf(" - %-8s: %v\n", FlagFilesFull, files)
	log.Printf(" - %-8s: %v\n", FlagFilesetsFull, filesets)
	log.Printf(" - %-8s: %v\n", FlagNTokensFull, nTokens)
	log.Printf(" - %-8s: %v\n", FlagNCompletionsFull, nCompletions)
	log.Printf(" - %-8s: %v\n", FlagOutputTypeFull, outputType)
	log.Printf(" - %-8s: %v\n", FlagAIBackendFull, aiBackend)

	return Flags{
		Request:      request,
		Write:        write,
		Path:         path,
		Files:        files,
		Filesets:     filesets,
		NTokens:      nTokens,
		NCompletions: nCompletions,
		OutputType:   outputType,
		AIBackend:    aiBackend,
	}
}

// ApplyAIFlags Applies the necessary values from the given flags struct
// into the provided AI modules.
func ApplyAIFlags(flags Flags, conf *config.Config) {
	// FIXME: convert all fields in config to pointers, test for pointer null
	if flags.NTokens != 0 {
		switch {
		case conf.GPT3 != nil:
			conf.GPT3.GenerateParams.MaxTokens = int(flags.NTokens)
		case conf.GPTJ != nil:
			conf.GPTJ.GenerateParams.ResponseLength = flags.NTokens
		case conf.BLOOM != nil:
			conf.BLOOM.GenerateParams.MaxNewTokens = int(flags.NTokens)
		}
	}
	if flags.NCompletions != 0 {
		if conf.GPT3 != nil {
			conf.GPT3.GenerateParams.N = int(flags.NCompletions)
		}
	}
}

// PrepareRequest Processes the user input along with provided environment variables,
// creating a Request object which is used for context in further requests.
func PrepareRequest(cmd *cobra.Command) (*Request, error) {
	flags := ExtractFlags(cmd)
	// Handle --path by changing the working directory
	// so that every file name we refer to is relative to path
	if flags.Path != "" {
		if err := os.Chdir(flags.Path); err != nil {
			return nil, err
		}
	}

	// Load the config from file if it exists, but if it doesn't exist
	// we'll just use the defaults and continue without error.
	// Errors here might return if the file exists but is invalid.
	conf := config.Config{}
	if err := conf.Load(cmd); err != nil {
		return nil, err
	}
	ApplyAIFlags(flags, &conf)

	// load files
	fm := filemap.NewFilemap()
	if err := fm.LoadFiles(flags.Files); err != nil {
		log.Fatalf("error loading files: %s\n", err.Error())
	}
	if len(flags.Filesets) > 0 {
		log.Printf("loading filesets: %v\n", flags.Filesets)
	}
	if err := fm.LoadFilesets(flags.Filesets, conf, config.ConfigFile); err != nil {
		log.Fatalf("error loading filesets: %s\n", err.Error())
	}
	filemapText := fm.EncodeToInputText()

	conf.PrintAsJSON()

	// FIXME: create default config methods for these
	r := Request{
		Config:      conf,
		Filemap:     fm,
		FilemapText: filemapText,
		UserRequest: flags.Request,
		IsWrite:     flags.Write,
		OutputType:  flags.OutputType,
	}

	return &r, nil
}

// PrintOrWriteOut Accepts a request object and writes the contents of the filemap
// to the disk if specified, otherwise it prints to STDOUT.
func PrintOrWriteOut(r *Request) error {
	if r.IsWrite {
		err := r.Filemap.WriteUpdatesToFiles()
		if err != nil {
			return err
		}
		return nil
	}

	// TODO: print as redirectable / pipeable write stream
	fmOutput, err := r.Filemap.EncodeToInputTextFullPaths(r.OutputType)
	if err != nil {
		return err
	}
	stringOut := strings.ReplaceAll(fmOutput, "\\n", "\n")
	log.Printf("\n%s\n", stringOut)

	return nil
}

// AddRequestFlags Appends flags to the given command which are then used at the command-line.
func AddRequestFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(
		FlagRequestFull, FlagRequestShort, "",
		"Requested changes in natural language (empty request will surprise you!)",
	)

	cmd.Flags().BoolP(
		FlagWriteFull, FlagWriteShort, false,
		"Write changes to the repo files (if not set the patch is printed to stdout)",
	)

	cmd.Flags().StringP(
		FlagPathFull, FlagPathShort, ".",
		"Path to the root of the repo",
	)

	cmd.Flags().StringP(
		FlagOutputTypeFull, FlagOutputTypeShort, "json",
		"How to format output",
	)

	cmd.Flags().StringP(
		FlagAIBackendFull, FlagAIBackendShort, string(ai.GPT3), "AI Backend to use",
	)
}

// // applyFlags Applies the appropriate configurations to the given config object
// // based on the provided flags.
// func applyFlags(conf *config.Config, flags *pflag.FlagSet) {
// 	nTokens, _ := flags.GetInt32(FlagNTokensFull)
// 	nCompletions, _ := flags.GetInt32(FlagNCompletionsFull)
// 	AIURL, _ := flags.GetString(FlagAIURLFull)
// 	backendS, _ := flags.GetString(FlagAIBackendFull)
// 	backend := ai.Backend(backendS)
// 	if nTokens != 0 {
// 		switch {
// 		case conf.GPT3 != nil:
// 			conf.GPT3.GenerateParams.MaxTokens = int(nTokens)
// 		case conf.GPTJ != nil:
// 			conf.GPTJ.GenerateParams.ResponseLength = nTokens
// 		case conf.BLOOM != nil:
// 			conf.BLOOM.GenerateParams.MaxNewTokens = int(nTokens)
// 		}
// 	}
// 	if nCompletions != 0 {
// 		if conf.GPT3 != nil {
// 			conf.GPT3.GenerateParams.N = int(nCompletions)
// 		}
// 	}
// 	// set flag for selected backend
// 	if AIURL != "" {
// 		switch backend {
// 		case ai.BLOOM:
// 			conf.BLOOM.URL = AIURL
// 		case ai.GPT3:
// 			conf.GPT3.BaseURL = AIURL
// 		case ai.GPTJ:
// 			conf.GPTJ.URL = AIURL
// 		}
// 	}
// }
