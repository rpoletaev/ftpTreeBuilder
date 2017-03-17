package ftpTreeBuilder

import (
	"fmt"
	"strconv"
	"strings"
)

func IsDir(name string) bool {
	return name != "" && !strings.Contains(name, ".")
}

func IsZip(name string) bool {
	return strings.Contains(name, ".zip")
}

func SizeFromFileString(s string) (int, error) {
	fields := strings.Fields(s)
	if len(fields) < 5 {
		return 0, fmt.Errorf("Неверный формат строки: %s", s)
	}

	sizeField := fields[4]
	if sizeField == "" {
		return 0, fmt.Errorf("Неверный формат данных в поле размер")
	}

	return strconv.Atoi(sizeField)
}

func NameFromFileSting(s string) (string, error) {
	fields := strings.Fields(s)
	if len(fields) < 9 {
		return "", fmt.Errorf("Неверный формат строки: %s", s)
	}

	nameField := fields[8]
	if nameField == "" {
		return "", fmt.Errorf("Неверный формат данных в поле имени")
	}

	return nameField, nil
}
