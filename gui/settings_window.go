package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type settingsWindow struct {
	e                *EEPGui
	w                fyne.Window
	ignoreError      *widget.Check
	readSliderLabel  *widget.Label
	readSlider       *widget.Slider
	writeSliderLabel *widget.Label
	writeSlider      *widget.Slider
}

func newSettingsWindow(e *EEPGui) *settingsWindow {
	w := e.app.NewWindow("Settings")
	sw := &settingsWindow{
		e:                e,
		w:                w,
		ignoreError:      widget.NewCheckWithData("Ignore read validation errors", e.state.ignoreError),
		readSliderLabel:  widget.NewLabel(""),
		readSlider:       widget.NewSliderWithData(20, 400, e.state.readDelayValue),
		writeSliderLabel: widget.NewLabel(""),
		writeSlider:      widget.NewSliderWithData(100, 400, e.state.writeDelayValue),
	}

	if f, err := sw.e.state.readDelayValue.Get(); err == nil {
		sw.readSliderLabel.SetText(delayLabel("Read", f))
	}

	if f, err := sw.e.state.writeDelayValue.Get(); err == nil {
		sw.writeSliderLabel.SetText(delayLabel("Write", f))
	}

	sw.ignoreError.OnChanged = func(b bool) {
		sw.e.app.Preferences().SetBool("ignore_read_errors", b)
	}

	sw.readSlider.OnChanged = func(f float64) {
		sw.readSliderLabel.SetText(delayLabel("Read", f))
		sw.e.app.Preferences().SetFloat("read_pin_delay", f)
		sw.e.state.readDelayValue.Set(f)
	}

	sw.writeSlider.OnChanged = func(f float64) {
		sw.writeSliderLabel.SetText(delayLabel("Write", f))
		sw.e.app.Preferences().SetFloat("write_pin_delay", f)
		sw.e.state.writeDelayValue.Set(f)
	}
	w.SetCloseIntercept(func() {
		w.Hide()
	})

	w.Resize(fyne.NewSize(300, 110))
	w.SetFixedSize(true)
	w.CenterOnScreen()
	w.SetContent(sw.layout())
	w.Hide()
	return sw
}

func (sw *settingsWindow) layout() fyne.CanvasObject {
	return container.NewVBox(
		sw.ignoreError,
		sw.readSliderLabel,
		sw.readSlider,
		sw.writeSliderLabel,
		sw.writeSlider,
	)
}

func delayLabel(t string, f float64) string {
	return fmt.Sprintf("%s Pin Delay: %.0f", t, f)
}
