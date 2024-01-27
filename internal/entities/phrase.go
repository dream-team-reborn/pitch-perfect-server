package entities

type Phrase struct {
	ID                 uint `gorm:"primarykey"`
	PlaceholdersAmount uint
}
