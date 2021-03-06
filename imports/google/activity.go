package google

import (
	"archive/zip"
	"encoding/json"
	"github.com/bionic-dev/bionic/imports/provider"
	"github.com/bionic-dev/bionic/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"os"
	"path/filepath"
)

const actionBatchSize = 100
const targetFilename = "MyActivity.json"

type Action struct {
	gorm.Model
	Header        string         `json:"header" gorm:"uniqueIndex:google_activity_key"`
	Title         string         `json:"title" gorm:"uniqueIndex:google_activity_key"`
	TitleURL      string         `json:"titleUrl"`
	Type          string         // Directory of the activity file. Such as "Google Analytics" and "Search".
	Time          types.DateTime `json:"time" gorm:"uniqueIndex:google_activity_key"`
	Products      []Product      `json:"products" gorm:"many2many:google_activity_products_assoc"`
	LocationInfos []LocationInfo `json:"locationInfos"`
	Subtitles     []Subtitle     `json:"subtitles"`
	Details       []Detail       `json:"details"`
}

func (Action) TableName() string {
	return tablePrefix + "activity"
}

type Product struct {
	gorm.Model
	Name string `gorm:"unique"`
}

func (Product) TableName() string {
	return tablePrefix + "activity_products"
}

func (p *Product) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	*p = Product{Name: str}
	return nil
}

type ActionProductAssoc struct {
	ActionID  int `gorm:"primaryKey;not null"`
	ProductID int `gorm:"primaryKey;not null"`
}

func (ActionProductAssoc) TableName() string {
	return tablePrefix + "activity_products_assoc"
}

type LocationInfo struct {
	gorm.Model
	ActionID  int
	Action    Action
	Name      string `json:"name"`
	URL       string `json:"url" `
	Source    string `json:"source"`
	SourceURL string `json:"sourceUrl"`
}

func (LocationInfo) TableName() string {
	return tablePrefix + "activity_location_infos"
}

type Subtitle struct {
	gorm.Model
	ActionID int
	Action   Action
	Name     string `json:"name"`
	URL      string `json:"url"`
}

func (Subtitle) TableName() string {
	return tablePrefix + "activity_subtitles"
}

type Detail struct {
	gorm.Model
	ActionID int
	Action   Action
	Name     string `json:"name"`
}

func (Detail) TableName() string {
	return tablePrefix + "activity_details"
}

func (p *google) importActivityFromArchive(inputPath string) error {
	r, err := zip.OpenReader(inputPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()

	for _, f := range r.File {
		filename := filepath.Base(f.Name)
		directory := filepath.Base(filepath.Dir(f.Name))
		if filename != targetFilename {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		if err := p.processActionsFile(rc, directory); err != nil {
			return err
		}
		if err := rc.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (p *google) importActivityFromDirectory(inputPath string) error {
	if !provider.IsPathDir(inputPath) {
		return nil
	}

	err := filepath.Walk(inputPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() != targetFilename {
				return nil
			}

			rc, err := os.Open(path)
			if err != nil {
				return err
			}

			err = p.processActionsFile(rc, filepath.Base(filepath.Dir(path)))
			if err != nil {
				return err
			}

			return nil
		})
	if err != nil {
		return err
	}
	return nil
}

func (p *google) processActionsFile(rc io.ReadCloser, directory string) error {
	decoder := json.NewDecoder(rc)
	if _, err := decoder.Token(); err != nil {
		return err
	} // Skip first token, which is opening the list

	var batch []Action

	for decoder.More() {
		var action Action
		err := decoder.Decode(&action)
		if err != nil {
			return err
		}

		action.Type = directory

		batch = append(batch, action)
		if len(batch) >= actionBatchSize {
			if err := p.saveActions(batch); err != nil {
				return err
			}
			batch = nil
		}
	}

	if err := p.saveActions(batch); err != nil {
		return err
	}

	return nil
}

func (p *google) saveActions(actions []Action) error {
	for i, action := range actions {
		for j, product := range action.Products {
			err := p.DB().
				FirstOrCreate(&actions[i].Products[j], map[string]interface{}{"name": product.Name}).
				Error
			if err != nil {
				return err
			}
		}
	}

	err := p.DB().
		Clauses(clause.OnConflict{
			DoNothing: true,
		}).
		CreateInBatches(actions, 1000).
		Error
	return err
}
