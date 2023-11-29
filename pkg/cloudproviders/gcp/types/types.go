package types

// DoneFunc should be called to after work is done to release internally held locks.
type DoneFunc func()
