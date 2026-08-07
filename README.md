# Agentic-CI

AI code review agent using Go CLI and Github Actions. WIP

## Done: 
- Cobra/Viper CLI with Finding + Report output contract
- go-gitdiff parser with own types and golden file tests
- Detector interface
- Secrets detector (using regex)
## Current: 
- Agent loop; includes tool calling loop, disposable worktree
## Planned additions:
- eval harness
- error handling detector
- retrevial
- Github actions + observability