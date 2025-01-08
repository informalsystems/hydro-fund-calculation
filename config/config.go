package config

type Config struct {
	TotalATOM       float64            `json:"totalATOM"`
	VenueFractions  map[string]float64 `json:"venueFractions"`
	ContractAddress string             `json:"contractAddress"`
	LCDURL          string             `json:"lcdURL"`
}

var GlobalConfig Config

func SetConfig(c Config) {
	GlobalConfig = c
}
