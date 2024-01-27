package entities

type Word struct {
	ID         uint `gorm:"primarykey"`
	CategoryId uint
}
