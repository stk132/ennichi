package main

import (
	"errors"
	"fmt"
	"github.com/fireworq/fireworq/model"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/stk132/tsutsu"
	"strconv"
)

var (
	MAIN_PAGE              = "main"
	FORM_PAGE              = "form"
	LABEL_QUEUE_NAME       = "queue name"
	LABEL_MAX_WORKERS      = "max workers"
	LABEL_POLLING_INTERVAL = "polling interval"
)

type app struct {
	root             *tview.Application
	pages            *tview.Pages
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

func (a *app) showQueueCreateForm() {
	clearPage := func() {
		a.pages.ShowPage(MAIN_PAGE)
		a.queueList.focus()
		a.pages.RemovePage(FORM_PAGE)
	}
	form := tview.NewForm().
		AddInputField(LABEL_QUEUE_NAME, "", 20, nil, nil).
		AddInputField(LABEL_MAX_WORKERS, "", 3, nil, nil).
		AddInputField(LABEL_POLLING_INTERVAL, "", 4, nil, nil)
	form.
		AddButton("Create", func() {
			a.logger.Info().Msg("created")

			queueNameItem := form.GetFormItemByLabel(LABEL_QUEUE_NAME)
			queueNameInput, ok := queueNameItem.(*tview.InputField)
			if !ok {
				a.logger.Err(errors.New("type assertion failed. FormItem to InputField"))
				clearPage()
				return
			}

			maxWorkersItem := form.GetFormItemByLabel(LABEL_MAX_WORKERS)
			maxWorkersInput, ok := maxWorkersItem.(*tview.InputField)
			if !ok {
				a.logger.Err(errors.New("type assertion failed. FormItem to InputField"))
				clearPage()
				return
			}

			pollingIntervalItem := form.GetFormItemByLabel(LABEL_POLLING_INTERVAL)
			pollingIntervalInput, ok := pollingIntervalItem.(*tview.InputField)
			if !ok {
				a.logger.Err(errors.New("type assertion failed. FormItem to InputField"))
				clearPage()
				return
			}

			maxWorkers, err := strconv.Atoi(maxWorkersInput.GetText())
			if err != nil {
				a.logger.Err(err)
				clearPage()
				return
			}

			pollingInterval, err := strconv.Atoi(pollingIntervalInput.GetText())
			if err != nil {
				a.logger.Err(err)
				clearPage()
				return
			}

			if _, err := a.client.CreateQueue(queueNameInput.GetText(), uint(pollingInterval), uint(maxWorkers)); err != nil {
				a.logger.Err(err)
				clearPage()
				return
			}

			a.logger.Info().Msg(fmt.Sprintf("queue_name: %s created", queueNameInput.GetText()))
			if err := a.refreshQueueList(); err != nil {
				a.logger.Err(err)
			}
			clearPage()
		}).
		AddButton("Cancel", func() {
			clearPage()
		})
	form.SetBorder(true).SetTitle("queue create form")
	a.pages.AddAndSwitchToPage("form", form, true)
	a.root.SetFocus(form)
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
	a.logger.Info().Msg("queue list refreshed")
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
	a.logger = zerolog.New(a.logWindow).With().Timestamp().Logger()
	a.jobCategoryTable = newJobCategoryTable(a)
	a.queueList = newQueueList(a)
	a.queueList.init()
	a.queueInfoTable = NewQueueInfoTable(a)

	a.root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'q':
			a.queueList.focus()
			return nil
		case 'l':
			a.logWindow.focus()
			return nil
		case 'n':
			a.showQueueCreateForm()
			return nil
		default:
			return event
		}
	})

	infoFlex.AddItem(a.queueInfoTable.table, 0, 1, false)
	infoFlex.AddItem(a.jobCategoryTable.table, 0, 4, false)
	flex.AddItem(a.queueList.list, 0, 1, true)
	flex.AddItem(infoFlex, 0, 1, false)
	globalFlex.AddItem(flex, 0, 5, true)
	globalFlex.AddItem(a.logWindow.textView, 0, 1, false)
	a.pages = tview.NewPages().AddAndSwitchToPage(MAIN_PAGE, globalFlex, true)

	return a.root.SetRoot(a.pages, true).Run()
}
