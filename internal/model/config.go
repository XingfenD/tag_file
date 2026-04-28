package model

type Config struct {
	ConfigID    uint   `gorm:"primaryKey;column:config_id" json:"config_id"`
	ConfigName  string `gorm:"column:config_name;size:100;not null" json:"config_name"`
	ConfigValue string `gorm:"column:config_value;type:text;not null" json:"config_value"`
	Description string `gorm:"column:description;type:text" json:"description"`
}

func (Config) TableName() string {
	return "config"
}
