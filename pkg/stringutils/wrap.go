package stringutils

// Wrap is a no-op that returns the text as-is. Line wrapping is unnecessary
// for server-side output (admission-control rejection messages, CLI reports) —
// kubectl and terminals handle their own wrapping.
func Wrap(text string) string {
	return text
}
