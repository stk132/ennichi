package main

import (
	"errors"
	"fmt"
	"github.com/fireworq/fireworq/model"
	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-isatty"
	"github.com/rivo/tview"
	"github.com/rs/zerolog"
	"github.com/stk132/tsutsu"
	"io"
	"os"
	"strconv"
)

var (
	MAIN_PAGE              = "main"
	QUEUE_FORM_PAGE        = "form"
	ROUTING_FORM_PAGE      = "routing"
	MODAL_PAGE             = "modal"
	LABEL_JOB_CATEGORY     = "job category"
	LABEL_QUEUE_NAME       = "queue name"
	LABEL_QUEUE_NAME_LIST  = "select queue name"
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

func (a *app) showRoutingCreateForm() {
	clearPage := func() {
		a.pages.ShowPage(MAIN_PAGE)
		a.queueList.focus()
		a.pages.RemovePage(ROUTING_FORM_PAGE)
	}

	showErrorModal := func(errorMessage string) {
		errorModal := tview.NewModal().
			SetText(errorMessage).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(ROUTING_FORM_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})

		a.pages.AddAndSwitchToPage(MODAL_PAGE, errorModal, true)
	}

	queueNames := make([]string, len(a.queues))
	for i, v := range a.queues {
		queueNames[i] = v.Name
	}
	form := tview.NewForm().
		AddDropDown(LABEL_QUEUE_NAME_LIST, queueNames, 0, func(option string, optionIndex int) {

		}).
		AddInputField(LABEL_JOB_CATEGORY, "", 20, nil, nil)

	form.
		AddButton("Create", func() {
			defer clearPage()
			queueNamesItem := form.GetFormItemByLabel(LABEL_QUEUE_NAME_LIST)
			queueNamesDropdown, ok := queueNamesItem.(*tview.DropDown)
			if !ok {
				err := errors.New("type assertion error. FormItem to Dropdown")
				a.logger.Err(err)
				showErrorModal(err.Error())
				return
			}

			_, selectedQueueName := queueNamesDropdown.GetCurrentOption()

			jobCategoryItem := form.GetFormItemByLabel(LABEL_JOB_CATEGORY)
			jobCategoryInput, ok := jobCategoryItem.(*tview.InputField)
			if !ok {
				err := errors.New("type assertion error. FormItem to InputField")
				a.logger.Err(err)
				showErrorModal(err.Error())
				return
			}

			jobCategory := jobCategoryInput.GetText()

			if _, err := a.client.CreateRouting(jobCategory, selectedQueueName); err != nil {
				a.logger.Err(err)
				showErrorModal(err.Error())
				return
			}

			a.logger.Info().Fields(map[string]interface{}{
				"job_category": jobCategory,
				"queue_name":   selectedQueueName,
			}).Msg("routing created")

			if err := a.refreshQueueList(); err != nil {
				a.logger.Err(err)
			}

			completeModal := tview.NewModal().
				SetText(fmt.Sprintf("routing created.\njob_category: %s \nqueue_name: %s", jobCategory, selectedQueueName)).
				AddButtons([]string{"Close"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					clearPage()
					a.pages.RemovePage(MODAL_PAGE)
				})

			a.pages.AddAndSwitchToPage(MODAL_PAGE, completeModal, true)
		}).
		AddButton("Cancel", func() {
			clearPage()
		})

	form.SetBorder(true).SetTitle("routing create form")
	a.pages.AddAndSwitchToPage(ROUTING_FORM_PAGE, form, true)
	a.root.SetFocus(form)
}

func (a *app) showQueueCreateForm() {
	clearPage := func() {
		a.pages.ShowPage(MAIN_PAGE)
		a.queueList.focus()
		a.pages.RemovePage(QUEUE_FORM_PAGE)
	}

	showErrorModal := func(errorMessage string) {
		errorModal := tview.NewModal().
			SetText(errorMessage).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(QUEUE_FORM_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})
		a.pages.AddAndSwitchToPage(MODAL_PAGE, errorModal, true)
	}

	form := tview.NewForm().
		AddInputField(LABEL_QUEUE_NAME, "", 20, nil, nil).
		AddInputField(LABEL_MAX_WORKERS, "", 3, nil, nil).
		AddInputField(LABEL_POLLING_INTERVAL, "", 4, nil, nil)
	form.
		AddButton("Create", func() {
			a.logger.Info().Msg("created")
			typeAssertionErr := errors.New("type assertion failed. FormItem to InputField")

			queueNameItem := form.GetFormItemByLabel(LABEL_QUEUE_NAME)
			queueNameInput, ok := queueNameItem.(*tview.InputField)
			if !ok {
				a.logger.Err(typeAssertionErr)
				showErrorModal(typeAssertionErr.Error())
				return
			}

			maxWorkersItem := form.GetFormItemByLabel(LABEL_MAX_WORKERS)
			maxWorkersInput, ok := maxWorkersItem.(*tview.InputField)
			if !ok {
				a.logger.Err(typeAssertionErr)
				showErrorModal(typeAssertionErr.Error())
				return
			}

			pollingIntervalItem := form.GetFormItemByLabel(LABEL_POLLING_INTERVAL)
			pollingIntervalInput, ok := pollingIntervalItem.(*tview.InputField)
			if !ok {
				a.logger.Err(typeAssertionErr)
				showErrorModal(typeAssertionErr.Error())
				return
			}

			maxWorkers, err := strconv.Atoi(maxWorkersInput.GetText())
			if err != nil {
				a.logger.Err(err)
				showErrorModal("please input number to max workers field")
				return
			}

			pollingInterval, err := strconv.Atoi(pollingIntervalInput.GetText())
			if err != nil {
				a.logger.Err(err)
				showErrorModal("please input number to polling interval field")
				return
			}

			if _, err := a.client.CreateQueue(queueNameInput.GetText(), uint(pollingInterval), uint(maxWorkers)); err != nil {
				a.logger.Err(err)
				showErrorModal(err.Error())
				return
			}

			a.logger.Info().Msg(fmt.Sprintf("queue_name: %s created", queueNameInput.GetText()))
			if err := a.refreshQueueList(); err != nil {
				a.logger.Err(err)
			}

			completeModal := tview.NewModal().
				SetText(fmt.Sprintf("queue: %s created", queueNameInput.GetText())).
				AddButtons([]string{"Close"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					clearPage()
					a.pages.RemovePage(MODAL_PAGE)
				})

			a.pages.AddAndSwitchToPage(MODAL_PAGE, completeModal, true)

		}).
		AddButton("Cancel", func() {
			clearPage()
		})
	form.SetBorder(true).SetTitle("queue create form")
	a.pages.AddAndSwitchToPage(QUEUE_FORM_PAGE, form, true)
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
	var writer io.Writer
	if isatty.IsTerminal(os.Stdout.Fd()) {
		writer = a.logWindow
	} else {
		writer = io.MultiWriter(a.logWindow, os.Stdout)
	}
	a.logger = zerolog.New(writer).With().Timestamp().Logger()
	a.jobCategoryTable = newJobCategoryTable(a)
	a.queueList = newQueueList(a)
	a.queueList.init()
	a.queueInfoTable = NewQueueInfoTable(a)

	a.root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ:
			a.queueList.focus()
			return nil
		case tcell.KeyCtrlL:
			a.logWindow.focus()
			return nil
		case tcell.KeyCtrlN:
			a.showQueueCreateForm()
			return nil
		case tcell.KeyCtrlJ:
			a.showRoutingCreateForm()
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
