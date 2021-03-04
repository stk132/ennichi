package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stk132/tsutsu"
	"log"
)

type app struct {
	root *tview.Application
	host string
}

func newApp(host string) *app {
	return &app{root: tview.NewApplication(), host: host}
}

func (a *app) run() error {
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	infoFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	client := tsutsu.NewTsutsu(a.host)
	routings, err := client.Routings()
	if err != nil {
		log.Fatal(err)
	}

	queues, err := client.Queues()
	if err != nil {
		log.Fatal(err)
	}

	routingMap := map[string][]string{}
	for _, v := range routings {
		if jobCategories, ok := routingMap[v.QueueName]; !ok {
			newJobCategories := []string{v.JobCategory}
			routingMap[v.QueueName] = newJobCategories
		} else {
			newJobCategories := append(jobCategories, v.JobCategory)
			routingMap[v.QueueName] = newJobCategories
		}
	}

	list := tview.NewList()
	list.SetBorder(true).SetTitle("queue list")

	queueInfoTable := NewQueueInfoTable(a)

	table := tview.NewTable()
	table.SetBorder(true).SetTitle("job categories")
	table.SetCell(0, 0, tview.NewTableCell("not selected"))
	for _, v := range queues {
		queueName := v.Name
		list.AddItem(v.Name, "", 'a', func() {
			table.Clear()
			if jobCategories, ok := routingMap[queueName]; ok {
				for i, jobCategory := range jobCategories {
					table.SetCell(i, 0, tview.NewTableCell(jobCategory))
				}
			}

			queue, err := client.Queue(queueName)
			if err != nil {
				log.Fatal(err)
			}
			queueInfoTable.setQueueInfo(queue)

		})
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'r':
			newQueues, err := client.Queues()
			if err != nil {
				log.Println(err)
				return nil
			}
			list.Clear()
			for _, v := range newQueues {
				list.AddItem(v.Name, "", 'a', nil)
			}
			return nil
		default:
			return event
		}

	})

	infoFlex.AddItem(queueInfoTable.table, 0, 1, false)
	infoFlex.AddItem(table, 0, 4, false)
	flex.AddItem(list, 0, 1, true)
	flex.AddItem(infoFlex, 0, 1, false)

	return a.root.SetRoot(flex, true).Run()
}
