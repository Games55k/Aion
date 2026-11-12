package theme

import "github.com/egoist/mygo/ui"

var (
	lightTheme = lightTokens.Theme()
	darkTheme  = darkTokens.Theme()
)

func Apply(c *ui.Context) {
	if c.Theme().Dark {
		c.SetTheme(darkTheme)
	} else {
		c.SetTheme(lightTheme)
	}
}

func TokensOf(c *ui.Context) *Tokens {
	if c.Theme().Dark {
		return Dark()
	}
	return Light()
}
