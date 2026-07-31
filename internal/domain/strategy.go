package domain

type Strategy interface {
	Traverse(quote Quote)
	Name() string
	Operations() []Operation
}
