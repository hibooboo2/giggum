package main

import "flag"

type CLIArgs struct {
	help         bool
	iterations   int
	verbose      bool
	debug        bool
	backup       bool
	restore      bool
	agentType    string
	timeout      int
	showProgress bool
	multiAgent   bool
	taskCommand  string
}

func parseFlags() CLIArgs {
	var args CLIArgs
	flag.BoolVar(&args.help, "h", false, "Show help message")
	flag.BoolVar(&args.help, "help", false, "Show help message")
	flag.IntVar(&args.iterations, "n", 10, "Number of iterations to run")
	flag.BoolVar(&args.verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&args.debug, "debug", false, "Enable debug output (implies -v)")
	flag.BoolVar(&args.backup, "backup", false, "Backup progress before running")
	flag.BoolVar(&args.restore, "restore", false, "Restore progress from latest backup and exit")
	flag.StringVar(&args.agentType, "agent", "backend-developer", "Specify agent type (tester, debugger, researcher, backend-developer, frontend-developer, ux, ui, marketer, feedbackseeker, simplifier, documentationwriter). Agent-only execution is enforced.")
	flag.IntVar(&args.timeout, "timeout", 30, "Agent timeout in minutes (default: 30)")
	flag.BoolVar(&args.showProgress, "show-progress", false, "Show agent progress for current project")
	flag.BoolVar(&args.multiAgent, "multi-agent", false, "Run coordinated multi-agent session")
	flag.StringVar(&args.taskCommand, "task", "", "Task command (list, create, add, remove, stats, import)")
	flag.Parse()
	return args
}
