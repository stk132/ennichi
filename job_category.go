package main

import "github.com/rivo/tview"

type JobCategoryTable struct {
	root  *app
	table *tview.Table
}

func newJobCategoryTable(root *app) *JobCategoryTable {
	table := tview.NewTable()
	table.SetBorder(true).SetTitle("job categories")
	table.SetCell(0, 0, tview.NewTableCell("not selected"))
	return &JobCategoryTable{
		root:  root,
		table: table,
	}
}

func (j *JobCategoryTable) draw(queueName string) error {
	j.table.Clear()
	if jobCategories, ok := j.root.routingMap[queueName]; ok {
		for i, jobCategory := range jobCategories {
			j.table.SetCell(i, 0, tview.NewTableCell(jobCategory))
		}
	}
	return nil
}
