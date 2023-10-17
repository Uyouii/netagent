package common

type AgentMode int

const (
	AGENT_MODE_SERVER AgentMode = 1 + iota
	AGENT_MODE_CLIENT
)
