package types

type ContextBudgetInput struct {
	ContextWindow int
	OutputLimit   int
	DeepAnalysis  bool
}

type ContextBudget struct {
	OutputReserve       int
	SystemReserve       int
	SafetyReserve       int
	Remaining           int
	RetrievalTarget     int
	RetrievalMultiplier float64
}
