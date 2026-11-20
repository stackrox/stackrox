package gocache

import "log"

// actionOutputID reads the outputID recorded for an action.
func actionOutputID(cacheDir, actionIDHex string) string {
	out, _, ok := readActionRecord(cacheDir, actionIDHex)
	if !ok {
		return ""
	}
	return out
}

func logRemoteIntegrity(actionIDHex, outputIDHex string) {
	log.Printf("cimemo-gocache: INTEGRITY(remote get) %s: body hash != %s — treating as miss",
		actionIDHex[:12], outputIDHex[:12])
}
