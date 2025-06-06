package template

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// XLSXFiller реализует TemplateFiller для шаблонов xlsx.
type XLSXFiller struct{}

// NewXLSX возвращает заполнитель XLSX.
func NewXLSX() XLSXFiller { return XLSXFiller{} }

// Fill возвращает содержимое шаблона без изменений.
// Fill заполняет файл Excel данными из SQL-запросов.
// Ожидается, что первая строка шаблона содержит имена столбцов в формате
// `{{column}}`. Эта строка остаётся заголовком, а под ней добавляются
// полученные данные.
func (XLSXFiller) Fill(tmpl []byte, data []map[string]any) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(tmpl))
	if err != nil {
		return nil, err
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("empty workbook")
	}
	sheet := sheets[0]

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("template missing header row")
	}

	header := rows[0]
	// Убираем фигурные скобки в заголовке.
	for j, cell := range header {
		name := strings.Trim(cell, "{}")
		addr, _ := excelize.CoordinatesToCellName(j+1, 1)
		f.SetCellValue(sheet, addr, name)
	}

	for i, row := range data {
		for j, col := range header {
			key := strings.Trim(col, "{}")
			if val, ok := row[key]; ok {
				addr, _ := excelize.CoordinatesToCellName(j+1, i+2)
				f.SetCellValue(sheet, addr, val)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
