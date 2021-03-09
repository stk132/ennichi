package main

import (
	"github.com/fireworq/fireworq/model"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/stk132/tsutsu"
)

type app struct {
	root             *tview.Application
	queueList        *QueueList
	queueInfoTable   *QueueInfoTable
	jobCategoryTable *JobCategoryTable
	logWindow        *LogWindow
	host             string
	client           *tsutsu.Tsutsu
	routings         []model.Routing
	queues           []model.Queue
	routingMap       map[string][]string
	logger           zerolog.Logger
}

func newApp(host string) *app {
	return &app{root: tview.NewApplication(), host: host, client: tsutsu.NewTsutsu(host)}
}

func (a *app) fetchData() error {
	routings, err := a.client.Routings()
	if err != nil {
		return err
	}

	queues, err := a.client.Queues()
	if err != nil {
		return err
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

	a.routings = routings
	a.queues = queues
	a.routingMap = routingMap
	return nil
}

func (a *app) drawJobCategory(queueName string) error {
	return a.jobCategoryTable.draw(queueName)
}

func (a *app) drawQueueInfo(queue model.Queue) {
	a.queueInfoTable.setQueueInfo(queue)
}

func (a *app) refreshQueueList() error {
	if err := a.fetchData(); err != nil {
		return err
	}
	a.queueList.clear()
	a.queueList.init()
	return nil
}

func (a *app) run() error {
	globalFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	infoFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	if err := a.fetchData(); err != nil {
		return err
	}

	a.logWindow = newLogWindow(a, 500)
	a.logger = zerolog.New(a.logWindow)
	a.jobCategoryTable = newJobCategoryTable(a)
	a.queueList = newQueueList(a)
	a.queueList.init()
	a.queueInfoTable = NewQueueInfoTable(a)

	infoFlex.AddItem(a.queueInfoTable.table, 0, 1, false)
	infoFlex.AddItem(a.jobCategoryTable.table, 0, 4, false)
	flex.AddItem(a.queueList.list, 0, 1, true)
	flex.AddItem(infoFlex, 0, 1, false)
	globalFlex.AddItem(flex, 0, 5, true)
	globalFlex.AddItem(a.logWindow.textView, 0, 1, false)

	return a.root.SetRoot(globalFlex, true).Run()
}
