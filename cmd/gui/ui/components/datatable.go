package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// DataColumn describes a single column header in a DataTable. Columns
// share equal width (driven by container.GridLayout); reorder columns
// to control visual emphasis.
type DataColumn struct {
	Header string
}

// DataRow is one row of data. Cells must have len == len(cols). Actions
// is an optional slice of widgets rendered to the right of the cells.
type DataRow struct {
	Cells   []string
	Actions []fyne.CanvasObject
}

// DataTable is a lightweight tabular view supporting per-row actions
// and a re-renderable row set. Build it once and call SetRows to swap
// data after a fetch.
type DataTable struct {
	cols      []DataColumn
	rowsBox   *fyne.Container
	canvas    fyne.CanvasObject
	emptyHint string
}

// NewDataTable returns a table with the given columns. emptyHint is
// shown when SetRows is called with no rows.
func NewDataTable(cols []DataColumn, emptyHint string) *DataTable {
	t := &DataTable{
		cols:      cols,
		rowsBox:   container.NewVBox(),
		emptyHint: emptyHint,
	}

	headerCells := make([]fyne.CanvasObject, len(cols))
	for i, c := range cols {
		headerCells[i] = widget.NewLabelWithStyle(c.Header, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	header := container.New(layout.NewGridLayout(len(cols)), headerCells...)

	t.canvas = container.NewBorder(
		container.NewVBox(header, widget.NewSeparator()),
		nil, nil, nil,
		container.NewVScroll(t.rowsBox),
	)
	t.SetRows(nil)
	return t
}

// CanvasObject returns the table's root widget for embedding.
func (t *DataTable) CanvasObject() fyne.CanvasObject { return t.canvas }

// SetRows replaces the displayed rows. Safe to call from any goroutine
// because the widget tree is updated synchronously; wrap the call in
// fyne.Do if invoked from a non-UI goroutine.
func (t *DataTable) SetRows(rows []DataRow) {
	t.rowsBox.RemoveAll()
	if len(rows) == 0 {
		hint := widget.NewLabel(t.emptyHint)
		hint.Alignment = fyne.TextAlignCenter
		t.rowsBox.Add(container.NewPadded(hint))
		t.rowsBox.Refresh()
		return
	}
	for _, r := range rows {
		t.rowsBox.Add(t.buildRow(r))
	}
	t.rowsBox.Refresh()
}

func (t *DataTable) buildRow(r DataRow) fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, len(t.cols))
	for i := range t.cols {
		text := ""
		if i < len(r.Cells) {
			text = r.Cells[i]
		}
		cells[i] = widget.NewLabel(text)
	}
	var row fyne.CanvasObject = container.New(layout.NewGridLayout(len(t.cols)), cells...)
	if len(r.Actions) > 0 {
		row = container.NewBorder(nil, nil, nil, container.NewHBox(r.Actions...), row)
	}
	return container.NewVBox(row, widget.NewSeparator())
}
