package repository

import (
	"encoding/json"
	"errors"
	iofs "io/fs"
	"os"
	"path/filepath"

	models "github.com/faust8888/go-musthave-metrics/internal/model"
)

// FileStorage wraps MemStorage and adds Save/Load to a JSON file.
// When syncWrite is true, every write immediately persists to disk.
type FileStorage struct {
	*MemStorage
	path      string
	syncWrite bool
}

func NewFileStorage(path string, syncWrite bool) *FileStorage {
	return &FileStorage{
		MemStorage: NewMemStorage(),
		path:       path,
		syncWrite:  syncWrite,
	}
}

func (fs *FileStorage) UpdateGauge(name string, value float64) {
	fs.MemStorage.UpdateGauge(name, value)
	if fs.syncWrite {
		_ = fs.Save()
	}
}

func (fs *FileStorage) UpdateCounter(name string, value int64) {
	fs.MemStorage.UpdateCounter(name, value)
	if fs.syncWrite {
		_ = fs.Save()
	}
}

func (fs *FileStorage) UpdateBatch(metrics []models.Metrics) error {
	_ = fs.MemStorage.UpdateBatch(metrics)
	if fs.syncWrite {
		return fs.Save()
	}
	return nil
}

func (fs *FileStorage) Save() error {
	gauges := fs.GetAllGauges()
	counters := fs.GetAllCounters()

	metrics := make([]models.Metrics, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := value
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Gauge, Value: &v})
	}
	for name, delta := range counters {
		d := delta
		metrics = append(metrics, models.Metrics{ID: name, MType: models.Counter, Delta: &d})
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(fs.path), "metrics-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, fs.path)
}

func (fs *FileStorage) Load() error {
	data, err := os.ReadFile(fs.path)
	if errors.Is(err, iofs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	for _, m := range metrics {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				fs.MemStorage.UpdateGauge(m.ID, *m.Value)
			}
		case models.Counter:
			if m.Delta != nil {
				fs.MemStorage.UpdateCounter(m.ID, *m.Delta)
			}
		}
	}
	return nil
}
