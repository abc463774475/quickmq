package base

type ServiceRouter interface {
	AddServer(server Service) error
	DelServer(id int64) error
	GetServer(id int64) Service

	GetID() int64
}
