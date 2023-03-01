package base

type Service interface {
	DelClient(id int64) error
	GetClient(id int64) AcceptClienHandler

	NewAcceptClienter(id int64) AcceptClienHandler

	GetID() int64
}
