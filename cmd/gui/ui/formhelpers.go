package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/teleconsys/DCS/cmd/gui/service"
	"github.com/teleconsys/DCS/cmd/gui/state"
	"github.com/teleconsys/DCS/cmd/gui/ui/feedback"
)

// Action runs a background job. Return a non-empty summary string on success
// for inline/banner feedback; leave it empty to fall back to RunSuccessBody.
type Action func(ctx context.Context, out io.Writer, snap state.ActorProfile) (summary string, err error)

// RunOpts configures RunButton completion feedback.
type RunOpts struct {
	OnSuccess   func(body string)
	OnError     func(err error)
	ResultLabel *widget.Label
}

// NewResultLabel returns a word-wrapped label for inline job results.
func NewResultLabel() *widget.Label {
	l := widget.NewLabel("")
	l.Wrapping = fyne.TextWrapWord
	return l
}

func setResultLabel(lbl *widget.Label, body string, kind feedback.Kind) {
	feedback.ApplyLabel(lbl, body, kind)
}

func showRunError(vc *ViewContext, lbl *widget.Label, err error, opts *RunOpts) {
	if opts != nil && opts.OnError != nil {
		opts.OnError(err)
		return
	}
	msg := err.Error()
	setResultLabel(lbl, msg, feedback.Error)
	if vc != nil && vc.Feedback != nil {
		vc.Feedback.Show(feedback.Error, msg)
	}
}

func showRunSuccess(vc *ViewContext, lbl *widget.Label, body string, opts *RunOpts) {
	if opts != nil && opts.OnSuccess != nil {
		opts.OnSuccess(body)
		return
	}
	if lbl != nil {
		setResultLabel(lbl, body, feedback.Success)
		return
	}
	if vc != nil && vc.Feedback != nil {
		vc.Feedback.Show(feedback.Success, feedback.FirstLine(body))
	}
}

// FormatCIDCreateResult builds a compact success message from the create pipeline result.
func FormatCIDCreateResult(r service.CIDCreateResult) string {
	msg := fmt.Sprintf("Content CID: %s\nObject ID: %s", r.CIDStr, r.CIDID)
	if strings.TrimSpace(r.CoinID) != "" {
		msg += "\nGas coin: " + r.CoinID
	}
	if r.EpochMS.End > 0 {
		msg += fmt.Sprintf("\nEpoch (ms): %d → %d", r.EpochMS.Start, r.EpochMS.End)
	}
	return msg
}

// FormatTxDigest builds a short success line when a service call returns a tx digest.
func FormatTxDigest(digest string) string {
	d := strings.TrimSpace(digest)
	if d == "" {
		return "Done."
	}
	return "Transaction digest:\n" + d
}

// FormatAccountNewResult summarizes a freshly written account file.
func FormatAccountNewResult(acc state.AccountFile) string {
	return fmt.Sprintf(
		"Account created.\n\nAlias: %s\nAddress: %s\n\nSaved to ./accounts/%s.json",
		acc.Alias, acc.Address, acc.Alias,
	)
}

// FormatAccountCoinsResult summarizes the coin listing RPC result.
func FormatAccountCoinsResult(r service.AccountCoinsResult) string {
	msg := fmt.Sprintf("Found %d coin object(s).", len(r.Coins))
	if strings.TrimSpace(r.BestIotaGas) != "" {
		msg += "\n\nRecommended IOTA gas coin:\n" + r.BestIotaGas
	}
	return msg
}

// RunButton builds a "Run" button that validates, runs via vc.Runner, and
// reports success/error through RunOpts (banner + optional ResultLabel).
func RunButton(vc *ViewContext, label, title string,
	validate func() error,
	action Action,
	opts ...RunOpts,
) *widget.Button {
	var o *RunOpts
	if len(opts) > 0 {
		o = &opts[0]
	}
	var resultLbl *widget.Label
	if o != nil {
		resultLbl = o.ResultLabel
	}

	var btn *widget.Button
	btn = widget.NewButton(label, func() {
		if validate != nil {
			if err := validate(); err != nil {
				showRunError(vc, resultLbl, err, o)
				return
			}
		}
		if resultLbl != nil {
			resultLbl.SetText("")
		}
		snap := vc.Snapshot()
		btn.Disable()
		var buf bytes.Buffer
		out := vc.TeeOutput(&buf)
		vc.Runner.Run(context.Background(), out, service.Job{
			Title: title,
			Fn: func(ctx context.Context, w io.Writer) (string, error) {
				return action(ctx, w, snap)
			},
		}, func(summary string, err error) {
			fyne.Do(func() {
				btn.Enable()
				captured := buf.String()
				if err != nil {
					showRunError(vc, resultLbl, err, o)
					return
				}
				body := strings.TrimSpace(summary)
				if body == "" {
					body = RunSuccessBody(title, label, captured)
					if strings.TrimSpace(body) == "" {
						body = fmt.Sprintf("%s finished.", label)
					}
				}
				showRunSuccess(vc, resultLbl, body, o)
			})
		})
	})
	return btn
}

// Card wraps a bold title above a vertical group of children.
func Card(title string, children ...fyne.CanvasObject) fyne.CanvasObject {
	header := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(append([]fyne.CanvasObject{header, widget.NewSeparator()}, children...)...)
}

// SectionHeader is a titled block for modal pages and settings sheets.
func SectionHeader(title, subtitle string) fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	box := container.NewVBox(titleLbl)
	if strings.TrimSpace(subtitle) != "" {
		sub := widget.NewLabel(subtitle)
		sub.Wrapping = fyne.TextWrapWord
		box.Add(sub)
	}
	return box
}

// SubsectionTitle is a smaller heading inside a section (e.g. wallet tools).
func SubsectionTitle(title string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// ModalScroll wraps modal body content with padding and vertical scroll.
func ModalScroll(inner fyne.CanvasObject) fyne.CanvasObject {
	return container.NewPadded(container.NewVScroll(TopBound(inner)))
}

// ActionRow right-aligns a primary action button under a form.
func ActionRow(btn fyne.CanvasObject) fyne.CanvasObject {
	return container.NewHBox(layout.NewSpacer(), btn)
}
