package symbols

import (
	"os"

	"github.com/grenmon2202/greninvestor/logging"
	"gopkg.in/yaml.v3"
)

var Nifty500YamlPath = "symbols/nifty500_symbols.yaml"
var SP500YamlPath = "symbols/snp500_symbols.yaml"

type Symbol struct {
	Name     string `yaml:"name"`
	Code     string `yaml:"code"`
	Exchange string `yaml:"exchange"`
	Currency string `yaml:"currency"`
}

type SymbolsStore struct {
	Symbols []Symbol `yaml:"symbols"`
}

var Nifty500SymbolsStore SymbolsStore
var SP500SymbolsStore SymbolsStore

func populateSymbolsNifty500() {
	data, err := os.ReadFile(Nifty500YamlPath)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(data, &Nifty500SymbolsStore); err != nil {
		panic(err)
	}
}

func populateSymbolsSP500() {
	data, err := os.ReadFile(SP500YamlPath)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(data, &SP500SymbolsStore); err != nil {
		panic(err)
	}
}

func RetrieveSymbolsNifty500() SymbolsStore {
	logging.Init()
	defer logging.L.Sync()

	if len(Nifty500SymbolsStore.Symbols) == 0 {
		logging.L.Info("Populating Nifty 500 symbols from YAML")
		populateSymbolsNifty500()
	}
	return Nifty500SymbolsStore
}

func RetrieveSymbolsSP500() SymbolsStore {
	logging.Init()
	defer logging.L.Sync()

	if len(SP500SymbolsStore.Symbols) == 0 {
		logging.L.Info("Populating S&P 500 symbols from YAML")
		populateSymbolsSP500()
	}
	return SP500SymbolsStore
}
