package prompt

// Choice is one selectable completion value with an optional description line.
type Choice struct {
	Value       string
	Display     string // picker column when set; Value is still what Pick returns
	Description string
}

func choiceLabel(c Choice) string {
	if c.Display != "" {
		return c.Display
	}
	return c.Value
}
