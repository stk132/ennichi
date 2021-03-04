package main

import (
	"fmt"
	"github.com/fireworq/fireworq/model"
	"github.com/rivo/tview"
	"sync"
)

type QueueInfoTable struct {
	root  *app
	table *tview.Table
	m     sync.Mutex
}

func NewQueueInfoTable(root *app) *QueueInfoTable {
	tbl := tview.NewTable()
	tbl.SetBorder(true).SetTitle("queue info")
	tbl.SetCell(0, 0, tview.NewTableCell("not selected"))
	return &QueueInfoTable{
		root:  root,
		table: tbl,
		m:     sync.Mutex{},
	}
}

func (q *QueueInfoTable) setQueueInfo(queue model.Queue) {
	q.m.Lock()
	defer q.m.Unlock()
	q.table.Clear()
	q.table.SetCell(0, 0, tview.NewTableCell(fmt.Sprintf("queue_name: %s", queue.Name)))
	q.table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("max_workers: %d", queue.MaxWorkers)))
	q.table.SetCell(2, 0, tview.NewTableCell(fmt.Sprintf("polling_interval: %d", queue.PollingInterval)))
}
