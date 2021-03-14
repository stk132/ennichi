package main

import (
	"fmt"
	"github.com/fireworq/fireworq/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sync"
)

type QueueList struct {
	root *app
	list *tview.List
}

func newQueueList(root *app) *QueueList {
	list := tview.NewList()
	list.SetBorder(true).SetTitle("queue list")
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'r':
				if err := root.refreshQueueList(); err != nil {
					root.logger.Err(err)
				}
				return nil
			case 'd':
				queueName, _ := list.GetItemText(list.GetCurrentItem())
				root.deleteQueue(queueName)
				return nil
			case 'e':
				queueName, _ := list.GetItemText(list.GetCurrentItem())
				root.showQueueEditForm(queueName)
				return nil
			default:
				return event
			}

		case tcell.KeyCtrlD:
			queueName, _ := list.GetItemText(list.GetCurrentItem())
			root.showDeleteRoutingForm(queueName)
			return nil
		default:
			return event
		}

	})
	return &QueueList{
		root: root,
		list: list,
	}
}

func (q *QueueList) focus() {
	q.root.root.SetFocus(q.list)
}

func (q *QueueList) clear() {
	q.list.Clear()
}

func (q *QueueList) init() {
	for _, v := range q.root.queues {
		queueName := v.Name
		q.list.AddItem(v.Name, "", 'a', func() {
			if err := q.root.drawJobCategory(queueName); err != nil {
				q.root.logger.Err(err)
			}

			queue, err := q.root.client.Queue(queueName)
			if err != nil {
				q.root.logger.Err(err)
			}
			q.root.drawQueueInfo(queue)
		})
	}
}

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
