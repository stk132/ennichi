package main

import (
	"encoding/json"
	"github.com/fireworq/fireworq/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	//box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)

	routings, err := getRoutingData()
	if err != nil {
		log.Fatal(err)
	}

	queues, err := getQueueData()
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
			newQueues, err := getQueueData()
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


func getQueueData() ([]model.Queue, error) {
	res, err := http.Get("http://localhost:18080/queues")
	if err != nil {
		return []model.Queue{}, err
	}

	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []model.Queue{}, err
	}

	var queues []model.Queue
	if err := json.Unmarshal(buf, &queues); err != nil {
		return queues, err
	}

	return queues, nil
}

func getRoutingData() ([]model.Routing, error) {
	res, err := http.Get("http://localhost:18080/routings")
	if err != nil {
		return []model.Routing{}, err
	}

	defer res.Body.Close()

	buf, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return []model.Routing{}, err
	}

	var routings []model.Routing

	if err := json.Unmarshal(buf, &routings); err != nil {
		return []model.Routing{}, err
	}

	return routings, nil
}