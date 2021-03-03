package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/stk132/tsutsu"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
)

var host = kingpin.Flag("host", "fireworq host url").Short('h').Required().String()

func main() {
	kingpin.Parse()
	//box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	client := tsutsu.NewTsutsu(*host)
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
	table := tview.NewTable()
	table.SetBorder(true).SetTitle("job categories")
	table.SetCell(0, 0, tview.NewTableCell("not selected"))
	for _, v := range queues {
		queueName := v.Name
		list.AddItem(v.Name, "", 'a', func(){
			table.Clear()
			if jobCategories, ok := routingMap[queueName]; ok {
				for i, jobCategory := range jobCategories {
					table.SetCell(i, 0, tview.NewTableCell(jobCategory))
				}
			}
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

	flex.AddItem(list, 0, 1, true)
	flex.AddItem(table, 0, 1, false)

	if err := tview.NewApplication().SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}