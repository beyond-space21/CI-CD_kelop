package videos

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// StringArray is a custom type for PostgreSQL text arrays
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	// PostgreSQL array format: {"item1","item2","item3"}
	// We need to escape quotes and format properly
	quoted := make([]string, len(a))
	for i, item := range a {
		// Escape quotes in the item
		escaped := strings.ReplaceAll(item, `"`, `""`)
		quoted[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}", nil
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("cannot scan non-string value into StringArray")
	}
	
	// PostgreSQL returns arrays in format: {item1,item2,item3}
	if len(bytes) > 0 && bytes[0] == '{' {
		// Remove braces
		str := string(bytes[1 : len(bytes)-1])
		if str == "" {
			*a = []string{}
			return nil
		}
		// Split by comma
		parts := []string{}
		current := ""
		inQuotes := false
		for i := 0; i < len(str); i++ {
			if str[i] == '"' {
				inQuotes = !inQuotes
			} else if str[i] == ',' && !inQuotes {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			} else {
				current += string(str[i])
			}
		}
		if current != "" {
			parts = append(parts, current)
		}
		*a = parts
		return nil
	}
	
	return json.Unmarshal(bytes, a)
}

type Videos struct {
	ID              int        `db:"id" json:"-"`
	VideoID         string     `db:"video_id" json:"video_id"`
	VideoURL        string     `db:"video_url" json:"video_url"`
	VideoThumbnail  string     `db:"video_thumbnail" json:"video_thumbnail"`
	VideoTitle      string     `db:"video_title" json:"video_title"`
	VideoDescription string    `db:"video_description" json:"video_description"`
	VideoTags       StringArray `db:"video_tags" json:"video_tags"`
	VideoViews      int        `db:"video_views" json:"video_views"`
	VideoUpvotes    int        `db:"video_upvotes" json:"video_upvotes"`
	VideoDownvotes  int        `db:"video_downvotes" json:"video_downvotes"`
	VideoComments   int        `db:"video_comments" json:"video_comments"`
	UserUID         string     `db:"user_uid" json:"user_uid"`
	UserUsername    string     `db:"user_username" json:"user_username"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}