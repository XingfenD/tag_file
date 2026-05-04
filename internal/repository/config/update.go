package config

import "github.com/XingfenD/tag_file/internal/model"

type UpdateOperation struct {
	repo *Repository
	cfg  *model.Config
	cols []string
}

func (r *Repository) Update(cfg *model.Config) *UpdateOperation {
	return &UpdateOperation{repo: r, cfg: cfg}
}

func (op *UpdateOperation) SetValue(v string) *UpdateOperation {
	op.cols = append(op.cols, "config_value")
	op.cfg.ConfigValue = v
	return op
}

func (op *UpdateOperation) Exec() error {
	if len(op.cols) == 0 {
		return op.repo.db.Save(op.cfg).Error
	}
	return op.repo.db.Model(op.cfg).Select(op.cols).Updates(*op.cfg).Error
}
