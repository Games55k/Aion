package theme

//go:generate bun ../../scripts/gen-tokens.ts

func Light() *Tokens {
	t := lightTokens
	return &t
}

func Dark() *Tokens {
	t := darkTokens
	return &t
}

func ColorTokenNames() []string {
	return append([]string(nil), colorTokenNames...)
}
