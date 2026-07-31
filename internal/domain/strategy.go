package domain

type Strategy interface {
	Traverse(candle Candle)
	Name() string
	Operations() []Operation
}
