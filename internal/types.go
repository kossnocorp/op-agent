package internal

type OpResponse struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Exit   int    `json:"exit"`
}

type HandshakeResponse struct {
	Version string `json:"version"`
	Whoami  string `json:"whoami"`
}
