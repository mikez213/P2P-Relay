package create

type NodeType int

const (
	Bootstrap NodeType = iota
	Relay
	MobileClient
	NodeRunner
)
