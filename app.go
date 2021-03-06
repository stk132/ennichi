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
	"sort"
	"strconv"
)

var (
	MAIN_PAGE                = "main"
	QUEUE_FORM_PAGE          = "form"
	EDIT_QUEUE_FORM_PAGE     = "edit form"
	ROUTING_FORM_PAGE        = "routing"
	DELETE_ROUTING_FORM_PAGE = "delete_routing"
	MODAL_PAGE               = "modal"
	LABEL_JOB_CATEGORY       = "job category"
	LABEL_QUEUE_NAME         = "queue name"
	LABEL_QUEUE_NAME_LIST    = "select queue name"
	LABEL_MAX_WORKERS        = "max workers"
	LABEL_POLLING_INTERVAL   = "polling interval"
	typeAssertionErr         = errors.New("type assertion failed. FormItem to InputField")
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
	sort.Slice(queues, func(i, j int) bool {
		return queues[i].Name < queues[j].Name
	})

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

func (a *app) showDeleteQueueErrorModal(queueName string) {
	errorMessageModal := tview.NewModal().
		SetText(fmt.Sprintf("delete queue failed. queue_name: %s", queueName)).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.SwitchToPage(MAIN_PAGE)
			a.pages.RemovePage(MODAL_PAGE)
		})

	a.pages.AddAndSwitchToPage(MODAL_PAGE, errorMessageModal, true)
}

func (a *app) showDeleteQueueSuccessModal(queueName string) {
	completeMessageModal := tview.NewModal().
		SetText(fmt.Sprintf("delete queue successed. queue_name: %s", queueName)).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			a.pages.SwitchToPage(MAIN_PAGE)
			a.pages.RemovePage(MODAL_PAGE)
		})

	a.pages.AddAndSwitchToPage(MODAL_PAGE, completeMessageModal, true)
}

func (a *app) showDeleteRoutingForm(queueName string) {
	routings, ok := a.routingMap[queueName]
	if !ok {
		noRoutingModal := tview.NewModal().
			SetText("routing not exists").
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(MAIN_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})
		a.pages.AddAndSwitchToPage(MODAL_PAGE, noRoutingModal, true)
		return
	}

	selectedJobCategory := routings[0]
	deleteRoutingForm := tview.NewForm().
		AddDropDown("job category", routings, 0, func(option string, optionIndex int) {
			selectedJobCategory = option
		}).AddButton("Delete", func() {
		if _, err := a.client.DeleteRouting(selectedJobCategory); err != nil {
			a.logger.Err(err)
			deleteFailedModal := tview.NewModal().
				SetText("delete routing failed. see log window.").
				AddButtons([]string{"Close"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					a.pages.SwitchToPage(MAIN_PAGE)
					a.pages.RemovePage(MODAL_PAGE)
				})
			a.pages.AddAndSwitchToPage(MODAL_PAGE, deleteFailedModal, true)
			return
		}
		a.logger.Info().Fields(map[string]interface{}{
			"job_category": selectedJobCategory,
			"queue_name":   queueName,
		}).Msg("routing deleted.")

		deleteSuccessModal := tview.NewModal().
			SetText(fmt.Sprintf("routing deleted. job_category: %s, queue_name: %s", selectedJobCategory, queueName)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(MAIN_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})

		if err := a.refreshQueueList(); err != nil {
			a.logger.Err(err)
		}

		a.pages.AddAndSwitchToPage(MODAL_PAGE, deleteSuccessModal, true)
	}).AddButton("Cancel", func() {
		a.pages.SwitchToPage(MAIN_PAGE)
		a.pages.RemovePage(DELETE_ROUTING_FORM_PAGE)
	})

	deleteRoutingForm.SetTitle("delete routing form")
	deleteRoutingForm.SetBorder(true)

	a.pages.AddAndSwitchToPage(DELETE_ROUTING_FORM_PAGE, deleteRoutingForm, true)
}

func (a *app) deleteQueue(queueName string) {
	deleteConfirmModal := tview.NewModal().
		SetText(fmt.Sprintf("Do you delete queue: %s ?", queueName)).
		AddButtons([]string{"No", "Yes"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				a.pages.SwitchToPage(MAIN_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			} else {
				if _, err := a.client.DeleteQueue(queueName); err != nil {
					a.logger.Err(err)
					a.showDeleteQueueErrorModal(queueName)
				}
				a.logger.Info().Fields(map[string]interface{}{
					"queue_name": queueName,
				}).Msg("queue deleted")

				if err := a.refreshQueueList(); err != nil {
					a.logger.Err(err)
				}
				a.showDeleteQueueSuccessModal(queueName)
			}
		})

	a.pages.AddAndSwitchToPage(MODAL_PAGE, deleteConfirmModal, true)
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

func (a *app) showQueueEditForm(queueName string) {
	queue, err := a.client.Queue(queueName)
	if err != nil {
		a.logger.Err(err)
		errorModal := tview.NewModal().
			SetText("queue information fetch failed.").
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(MAIN_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})
		a.pages.AddAndSwitchToPage(MODAL_PAGE, errorModal, true)
		return
	}

	clearPage := func() {
		a.pages.ShowPage(MAIN_PAGE)
		a.queueList.focus()
		a.pages.RemovePage(EDIT_QUEUE_FORM_PAGE)
	}

	showErrorModal := func(errorMessage string) {
		errorModal := tview.NewModal().
			SetText(errorMessage).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				a.pages.SwitchToPage(EDIT_QUEUE_FORM_PAGE)
				a.pages.RemovePage(MODAL_PAGE)
			})
		a.pages.AddAndSwitchToPage(MODAL_PAGE, errorModal, true)
	}

	editForm := tview.NewForm().
		AddInputField(LABEL_MAX_WORKERS, strconv.Itoa(int(queue.MaxWorkers)), 3, nil, nil).
		AddInputField(LABEL_POLLING_INTERVAL, strconv.Itoa(int(queue.PollingInterval)), 4, nil, nil)

	editForm.AddButton("Update", func() {
		maxWorkersItem := editForm.GetFormItemByLabel(LABEL_MAX_WORKERS)
		maxWorkersInput, ok := maxWorkersItem.(*tview.InputField)
		if !ok {
			a.logger.Err(typeAssertionErr)
			showErrorModal(typeAssertionErr.Error())
			return
		}

		pollingIntervalItem := editForm.GetFormItemByLabel(LABEL_POLLING_INTERVAL)
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

		if _, err := a.client.CreateQueue(queueName, uint(pollingInterval), uint(maxWorkers)); err != nil {
			a.logger.Err(err)
			showErrorModal(err.Error())
			return
		}

		a.logger.Info().Fields(map[string]interface{}{
			"queue_name":       queueName,
			"max_workers":      maxWorkers,
			"polling_interval": pollingInterval,
		}).Msg("queue updated.")

		if err := a.refreshQueueList(); err != nil {
			a.logger.Err(err)
		}

		completeModal := tview.NewModal().
			SetText(fmt.Sprintf("queue updated. queue_name: %s, max_workers: %d, polling_interval: %d", queueName, maxWorkers, pollingInterval)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				clearPage()
				a.pages.RemovePage(MODAL_PAGE)
			})

		a.pages.AddAndSwitchToPage(MODAL_PAGE, completeModal, true)
	}).AddButton("Cancel", func() {
		clearPage()
	})

	editForm.SetBorder(true).SetTitle(fmt.Sprintf("queue: %s edit form", queueName))

	a.pages.AddAndSwitchToPage(EDIT_QUEUE_FORM_PAGE, editForm, true)
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
			//typeAssertionErr := errors.New("type assertion failed. FormItem to InputField")

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
