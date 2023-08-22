package config

type Config struct {
	ChainsSupported map[string]*ChainConfig //chainName => chainConfig
	ChainPrefix     string
}

func DefaultConfig() Config {
	c := Config{
		ChainPrefix: "virtual ",
	}
	c.ChainsSupported = make(map[string]*ChainConfig)
	return c
}

func (c *Config) RegisterChainConfig(chainName string, chainConfig *ChainConfig) {
	c.ChainsSupported[chainName] = chainConfig
}

type ChainConfig struct {
	ChainName                   string
	ChainId                     [32]byte
	ClientUrls                  []string //format is: ip:port,username,password
	BlockInterval               int64
	MaxTxsInBlock               int
	GenesisMainChainBlockHeight int64
}

func NewBchChainConfig(config *Config, clientUrls []string, GenesisMainChainBlockHeight int64) *ChainConfig {
	c := &ChainConfig{
		ChainName:     config.ChainPrefix + "Bitcoin Cash",
		ClientUrls:    clientUrls,
		BlockInterval: 5, //5s
		MaxTxsInBlock: 2000,
	}
	c.ChainId = convertChainNameToChainId(c.ChainName)
	c.GenesisMainChainBlockHeight = GenesisMainChainBlockHeight
	return c
}

func convertChainNameToChainId(name string) (id [32]byte) {
	copy(id[:], []byte(name))
	return
}
