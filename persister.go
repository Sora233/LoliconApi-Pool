package loliconApiPool

type Persist interface {
	Load(r18Type R18Type) ([]*Setu, error)
	Store(R18Type, []*Setu) error
}

type NilPersist struct{}

func (*NilPersist) Load(r18Type R18Type) ([]*Setu, error) {
	return make([]*Setu, 0), nil
}

func (*NilPersist) Store(R18Type, []*Setu) error {
	return nil
}
func NewNilPersist() *NilPersist {
	return &NilPersist{}
}
