package query

import (
	"fmt"
	"strings"
)

// Validate проверяет SQL-запрос на наличие запрещённых конструкций.
func Validate(sql string) error {
	forbidden := []string{"DROP", "DELETE", "UPDATE", "INSERT", "CREATE", "ALTER"}
	upper := strings.ToUpper(sql)
	for _, f := range forbidden {
		if strings.Contains(upper, f) {
			return fmt.Errorf("forbidden operation: %s", f)
		}
	}
	return nil
}
