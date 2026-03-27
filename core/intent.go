package core

// Intent represents what kind of payment the service wants.
type Intent string

const (
	IntentCharge    Intent = "charge"
	IntentAuthorize Intent = "authorize"
	IntentSubscribe Intent = "subscribe"
	IntentMandate   Intent = "mandate"
)

// ValidIntents returns all valid intent values.
func ValidIntents() []Intent {
	return []Intent{IntentCharge, IntentAuthorize, IntentSubscribe, IntentMandate}
}

// IsValid reports whether the intent is a recognized value.
func (i Intent) IsValid() bool {
	switch i {
	case IntentCharge, IntentAuthorize, IntentSubscribe, IntentMandate:
		return true
	}
	return false
}
