package types

import (
	"encoding/json"
	"os"
)

type ContractEntrance struct {
	ContractAddress string
}

type ConflictItem struct {
	Left  []ContractEntrance
	Right []ContractEntrance
}

func LoadingConf(file string) ([]ConflictItem, error) {
	items := []ConflictItem{}
	filePtr, err := os.Open(file)
	if err != nil {
		return []ConflictItem{}, err
	}
	defer filePtr.Close()

	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&items)
	if err != nil {
		//fmt.Println("Decoder failed", err.Error())
		return items, err
	}
	return items, nil
}
