package config

type Driver interface {
	Exists() (bool, error)
	Write(config Config) error
	Read() (Config, error)
}

func NewStore(driver Driver) (Store, error) {
	exists, err := driver.Exists()
	if err != nil {
		return Store{}, err
	}
	if !exists {
		if err := driver.Write(defaultConfig); err != nil {
			return Store{}, err
		}
	}

	return Store{
		driver: driver,
	}, nil
}

type Store struct {
	driver Driver
}

func (p *Store) GetConfig() (Config, error) {
	return p.driver.Read()
}

func (p *Store) UpdateConfig(fn func(cfg Config) (Config, error)) error {
	cfg, err := p.driver.Read()
	if err != nil {
		return err
	}

	cfg, err = fn(cfg)
	if err != nil {
		return err
	}

	return p.driver.Write(cfg)
}
