package file

import (
	"time"
)

// SetCreateBy 设置创建人id
func (e *Model) SetCreateBy(createBy string) {
	e.CreateBy = createBy
}

// SetUpdateBy 设置修改人id
func (e *Model) SetUpdateBy(updateBy string) {
	e.UpdateBy = updateBy
}

type Model struct {
	Id         int64     `gorm:"primaryKey;autoIncrement;comment:主键编码"`
	CreateTime time.Time `json:"create_time" gorm:"comment:创建时间"`
	UpdateTime time.Time `json:"update_time" gorm:"comment:最后更新时间"`
	CreateBy   string    `gorm:"size:50;comment:创建者"`
	UpdateBy   string    `gorm:"size:50;comment:更新者"`
	Status     int       `gorm:"comment:更新者"`
}
