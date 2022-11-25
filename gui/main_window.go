package gui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	sdialog "github.com/sqweek/dialog"
)

type mainWindow struct {
	e *EEPGui
	w fyne.Window

	logList binding.StringList
	log     *widget.List

	rescanButton *widget.Button
	portList     *widget.Select

	editButton  *widget.Button
	viewButton  *widget.Button
	readButton  *widget.Button
	writeButton *widget.Button
	eraseButton *widget.Button

	progressBar *widget.ProgressBar
}

func newMainWindow(e *EEPGui) *mainWindow {
	window := e.app.NewWindow("Saab CIM Cloner by Hirschmann-Koxha GbR")
	window.SetMaster()
	m := &mainWindow{e: e, w: window}
	window.SetContent(m.layout())
	window.Resize(fyne.NewSize(1200, 600))
	window.Show()
	return m
}

func (m *mainWindow) layout() fyne.CanvasObject {
	logList := binding.NewStringList()
	m.logList = logList
	m.log = createLogList(logList)
	m.progressBar = widget.NewProgressBar()

	m.rescanButton = widget.NewButtonWithIcon("Rescan ports", theme.ViewRefreshIcon(), func() {
		m.portList.Options = m.listPorts()
	})

	m.portList = widget.NewSelect(m.listPorts(), func(s string) {
		m.e.state.port = s
		m.e.app.Preferences().SetString("port", s)
	})
	m.portList.Alignment = fyne.TextAlignCenter

	if m.e.state.port != "" {
		m.portList.PlaceHolder = m.e.state.port
	}
	m.editButton = widget.NewButtonWithIcon("Edit", theme.FileIcon(), func() {
		newEditWindow(m.e)
	})
	m.viewButton = widget.NewButtonWithIcon("View", theme.SearchIcon(), m.viewClickHandler)
	m.readButton = widget.NewButtonWithIcon("Read", theme.DownloadIcon(), m.readClickHandler)
	m.writeButton = widget.NewButtonWithIcon("Write", theme.UploadIcon(), m.writeClickHandler)
	m.eraseButton = widget.NewButtonWithIcon("Erase", theme.DeleteIcon(), m.eraseClickHandler)

	left := container.New(layout.NewMaxLayout(), m.log)
	right := container.NewVBox(
		m.rescanButton,
		m.portList,
		m.editButton,
		m.viewButton,
		m.readButton,
		m.writeButton,
		m.eraseButton,
		layout.NewSpacer(),
		widget.NewButtonWithIcon("Help", theme.HelpIcon(), func() {
			m.e.hw.w.Show()
		}),
		widget.NewButtonWithIcon("Copy log", theme.ContentCopyIcon(), func() {
			if content, err := m.logList.Get(); err == nil {
				m.w.Clipboard().SetContent(strings.Join(content, "\n"))
			}
		}),
		widget.NewButtonWithIcon("Clear log", theme.ContentClearIcon(), func() {
			m.logList.Set([]string{})
		}),
		widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), func() {
			m.e.sw.w.Show()
		}),
	)

	split := container.NewHSplit(left, right)
	split.Offset = 0.99
	view := container.NewVSplit(split, m.progressBar)
	view.Offset = 1
	return view
}

func (m *mainWindow) viewClickHandler() {
	m.viewButton.Disable()
	go func() {
		defer m.viewButton.Enable()
		filename, err := sdialog.File().Filter("Bin file", "bin").Title("Select file to view").Load()
		if err != nil {
			if err.Error() == "Cancelled" {
				return
			}
			m.output(err.Error())
			return
		}

		bin, err := os.ReadFile(filename)
		if err != nil {
			m.output(err.Error())
			return
		}
		newViewerWindow(m.e, filename, bin, false)
	}()
}

func (m *mainWindow) readClickHandler() {
	go func() {
		m.disableButtons()
		defer m.enableButtons()
		if m.e.state.port == "" {
			m.output("Please select a port first")
			return
		}

		ignoreReadErrors, _ := m.e.state.ignoreError.Get()

		rawBytes, bin, err := m.readCIM(m.e.state.port, 1)
		if err != nil {
			m.output(err.Error())
			if ignoreReadErrors {
				m.saveFile(rawBytes)
			} else {
				dialog.ShowConfirm("Error reading CIM", "There was errors reading, view anyway?", func(b bool) {
					if b {
						newViewerWindow(m.e, fmt.Sprintf("failed read from %s", time.Now().Format(time.RFC1123Z)), rawBytes, true)
					}
				}, m.w)
			}
			return
		}

		xorBytes, err := bin.XORBytes()
		if err != nil {
			m.output(err.Error())
			return
		}

		m.printKV("MD5", bin.MD5())
		m.printKV("CRC32", bin.CRC32())
		m.printKV("VIN", bin.Vin.Data)
		m.printKV("MY", bin.Vin.Data[9:10])
		m.printKV("End model (HW+SW)", fmt.Sprintf("%d%s", bin.PartNo1, bin.PartNo1Rev))
		m.printKV("Base model (HW+boot)", fmt.Sprintf("%d%s", bin.PnBase1, bin.PnBase1Rev))
		m.printKV("Delphi part number", fmt.Sprintf("%d", bin.DelphiPN))
		m.printKV("SAAB part number", fmt.Sprintf("%d", bin.PartNo))
		m.printKV("Configuration Version", fmt.Sprintf("%d", bin.ConfigurationVersion))

		newViewerWindow(m.e, fmt.Sprintf("successful read from %s", time.Now().Format(time.RFC1123Z)), xorBytes, true)
	}()
}

func (m *mainWindow) writeClickHandler() {
	go func() {
		if m.e.state.port == "" {
			m.output("Please select a port first")
			return
		}
		m.disableButtons()
		defer m.enableButtons()

		filename, bin, err := loadFile()
		if err != nil {
			if err.Error() == "Cancelled" {
				return
			}
			m.output(err.Error())
			return
		}

		ok := sdialog.Message("%s", "Do you want to continue?").Title("Are you sure?").YesNo()
		if !ok {
			return
		}
		m.output("Flashing CIM ... ")
		start := time.Now()
		if ok := m.writeCIM(m.e.state.port, bin); !ok {
			return
		}
		m.output("Flashed %s, took %s", filename, time.Since(start).String())
	}()
}

func (m *mainWindow) eraseClickHandler() {
	go func() {
		if m.e.state.port == "" {
			m.output("Please select a port first")
			return
		}
		m.disableButtons()
		defer m.enableButtons()

		ok := sdialog.Message("%s", "Do you want to erase CIM?").Title("Are you sure?").YesNo()
		if !ok {
			return
		}
		start := time.Now()
		sr, err := m.openPort(m.e.state.port)
		if sr != nil {
			defer sr.Close()
		}

		if err != nil {
			m.output("Failed to init adapter: %v", err)
			return
		}

		m.output("Erasing ... ")
		if err := m.erase(sr); err != nil {
			m.output(err.Error())
		}

		m.output("Erase took %s", time.Since(start).String())
	}()
}

func (m *mainWindow) saveFile(data []byte) bool {
	filename, err := sdialog.File().Filter("Bin file", "bin").Title("Save bin file").Save()
	if err != nil {
		if err.Error() == "Cancelled" {
			return false
		}
		m.output(err.Error())
		return false
	}
	filename = addSuffix(filename, ".bin")

	if err := os.WriteFile(filename, data, 0644); err == nil {
		m.output("Saved to %s", filename)
	} else {
		m.output(err.Error())
		return false
	}
	return true
}

func loadFile() (string, []byte, error) {
	filename, err := sdialog.File().Filter("Bin file", "bin").Title("Load bin file").Load()
	if err != nil {
		return "", nil, err
	}

	bin, err := os.ReadFile(filename)
	if err != nil {
		return "", nil, err
	}
	return filename, bin, nil
}

func addSuffix(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		return s + suffix
	}
	return s
}

func (m *mainWindow) printKV(k, v string) {
	m.output(k + ": " + v)
}

func createLogList(listData binding.StringList) *widget.List {
	return widget.NewListWithData(
		listData,
		func() fyne.CanvasObject {
			w := widget.NewLabel("")
			w.TextStyle.Monospace = true
			return w
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			i := item.(binding.String)
			txt, err := i.Get()
			if err != nil {
				panic(err)
			}
			obj.(*widget.Label).SetText(txt)
		},
	)
}

func (m *mainWindow) output(format string, values ...interface{}) int {
	var text string
	if format != "" {
		text = fmt.Sprintf("%s - %s", time.Now().Format("15:04:05.000"), fmt.Sprintf(format, values...))
	}
	m.logList.Append(text)
	m.log.Refresh()
	m.log.ScrollToBottom()
	return m.logList.Length()
}

func (m *mainWindow) append(format string, values ...interface{}) {
	di, err := m.logList.GetValue(m.logList.Length() - 1)
	if err != nil {
		panic(err)
	}
	m.logList.SetValue(m.logList.Length()-1, di+fmt.Sprintf(format, values...))
	m.log.Refresh()
}

func (m *mainWindow) disableButtons() {
	m.rescanButton.Disable()
	m.portList.Disable()
	//m.viewButton.Disable()
	m.readButton.Disable()
	m.writeButton.Disable()
	m.eraseButton.Disable()
}

func (m *mainWindow) enableButtons() {
	m.rescanButton.Enable()
	m.readButton.Enable()
	m.portList.Enable()
	//m.viewButton.Enable()
	m.readButton.Enable()
	m.writeButton.Enable()
	m.eraseButton.Enable()
}
