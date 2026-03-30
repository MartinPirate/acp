package core

// ContainsString checks if a string slice contains the given string.
func ContainsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ContainsIntent checks if an Intent slice contains the given intent.
func ContainsIntent(slice []Intent, i Intent) bool {
	for _, v := range slice {
		if v == i {
			return true
		}
	}
	return false
}

// ContainsCurrency checks if a Currency slice contains the given currency.
func ContainsCurrency(slice []Currency, c Currency) bool {
	for _, v := range slice {
		if v == c {
			return true
		}
	}
	return false
}

// HasStringOverlap returns true if slices a and b share at least one element.
func HasStringOverlap(a, b []string) bool {
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// HasIntentOverlap returns true if slices a and b share at least one Intent.
func HasIntentOverlap(a, b []Intent) bool {
	set := make(map[Intent]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// HasCurrencyOverlap returns true if slices a and b share at least one Currency.
func HasCurrencyOverlap(a, b []Currency) bool {
	set := make(map[Currency]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}
