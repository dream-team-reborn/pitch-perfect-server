package core

import (
	"encoding/json"
	"github.com/sourcegraph/conc/iter"
	"os"
	database "pitch-perfect-server/internal/db"
	"pitch-perfect-server/internal/entities"
)

func InitConfig() error {
	bytes, err := os.ReadFile("./config/1/game_configuration.json")
	if err != nil {
		return err
	}

	var data interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		panic(err)
	}
	dataMap := data.(map[string]interface{})

	phrases := dataMap["phrases"].([]interface{})
	iter.ForEach(phrases,
		func(subDataPtr *interface{}) {
			subData := (*subDataPtr).(map[string]interface{})
			id := uint(subData["id"].(float64))
			placeHolder := uint(subData["placeholdersAmount"].(float64))

			entity := entities.Phrase{ID: id, PlaceholdersAmount: placeHolder}
			database.Db.Save(&entity)
		})

	words := dataMap["words"].([]interface{})
	iter.ForEach(words,
		func(subDataPtr *interface{}) {
			subData := (*subDataPtr).(map[string]interface{})
			id := uint(subData["id"].(float64))
			category := uint(subData["categoryId"].(float64))

			entity := entities.Word{ID: id, CategoryId: category}
			database.Db.Save(&entity)
		})

	categories := dataMap["categories"].([]interface{})
	iter.ForEach(categories,
		func(subDataPtr *interface{}) {
			subData := (*subDataPtr).(map[string]interface{})
			id := uint(subData["id"].(float64))

			entity := entities.Category{ID: id}
			database.Db.Save(&entity)
		})

	return nil
}

func GetPhrases() ([]entities.Phrase, error) {
	var phrases []entities.Phrase
	tx := database.Db.Find(&phrases)
	return phrases, tx.Error
}

func GetWords() ([]entities.Word, error) {
	var words []entities.Word
	tx := database.Db.Find(&words)
	return words, tx.Error
}

func GetCategories() ([]entities.Category, error) {
	var categories []entities.Category
	tx := database.Db.Find(&categories)
	return categories, tx.Error
}
