package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func searchAndSort(c *gin.Context, q *gorm.DB, searchCols []string, sortCols map[string]string) (*gorm.DB, error) {
	if term := strings.TrimSpace(c.Query("q")); term != "" {
		pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(term) + "%"
		conds := make([]string, len(searchCols))
		args := make([]any, len(searchCols))
		for i, col := range searchCols {
			conds[i] = col + " ILIKE ?"
			args[i] = pattern
		}
		q = q.Where(strings.Join(conds, " OR "), args...)
	}

	if s := c.Query("sort"); s != "" {
		desc := strings.HasPrefix(s, "-")
		col, ok := sortCols[strings.TrimPrefix(s, "-")]
		if !ok {
			return nil, errInvalidSort
		}
		q = q.Order(clause.OrderByColumn{Column: clause.Column{Name: col}, Desc: desc})
	}
	return q.Order("id"), nil
}
