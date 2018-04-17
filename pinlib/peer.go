package pinlib

// Peer is an Interface which should implement Start method
// This is mainly used for making the code simpler if the both the client and
// servers are written in the same program.
type Peer interface {
	Start() error
}
