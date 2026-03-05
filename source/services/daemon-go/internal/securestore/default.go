package securestore

func NewDefaultManager() (*Manager, error) {
	return NewManager("personal-agent", "keyring", NewKeyringBackend())
}
