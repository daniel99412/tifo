package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	calHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Align(lipgloss.Center)

	calDayStyle = lipgloss.NewStyle().
			Width(3).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("250"))

	calWeekdayStyle = lipgloss.NewStyle().
			Width(3).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("240"))

	calSelectedStyle = lipgloss.NewStyle().
				Width(3).
				Align(lipgloss.Center).
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("39"))

	calTodayStyle = lipgloss.NewStyle().
			Width(3).
			Align(lipgloss.Center).
			Foreground(lipgloss.Color("39"))

	calNavStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	calBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63"))
)

type Calendar struct {
	cursorDate time.Time
	viewMonth  time.Time
}

func NewCalendar(cursorDate time.Time) Calendar {
	now := time.Now()
	return Calendar{
		cursorDate: cursorDate,
		viewMonth:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
	}
}

func (c *Calendar) SetDate(t time.Time) {
	c.cursorDate = t
	c.syncViewMonth()
}

func (c *Calendar) Date() time.Time {
	return c.cursorDate
}

func (c *Calendar) PrevMonth() {
	c.viewMonth = c.viewMonth.AddDate(0, -1, 1)
}

func (c *Calendar) NextMonth() {
	c.viewMonth = c.viewMonth.AddDate(0, 1, 1)
}

func (c *Calendar) PrevDay() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, -1)
}

func (c *Calendar) NextDay() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, 1)
}

func (c *Calendar) CursorUp() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, -7)
	c.syncViewMonth()
}

func (c *Calendar) CursorDown() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, 7)
	c.syncViewMonth()
}

func (c *Calendar) syncViewMonth() {
	y, m, _ := c.cursorDate.Date()
	vy, vm, _ := c.viewMonth.Date()
	if y != vy || m != vm {
		c.viewMonth = time.Date(y, m, 1, 0, 0, 0, 0, c.cursorDate.Location())
	}
}

func (c *Calendar) CursorLeft() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, -1)
	c.syncViewMonth()
}

func (c *Calendar) CursorRight() {
	c.cursorDate = c.cursorDate.AddDate(0, 0, 1)
	c.syncViewMonth()
}

func (c *Calendar) Render(width, height int) string {
	calW := 23
	calH := 10

	header := calHeaderStyle.Render(c.viewMonth.Format("January 2006"))

	weekdays := ""
	for _, d := range []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"} {
		weekdays += calWeekdayStyle.Render(d)
	}

	firstDay := time.Date(c.viewMonth.Year(), c.viewMonth.Month(), 1, 0, 0, 0, 0, c.viewMonth.Location())
	daysInMonth := time.Date(c.viewMonth.Year(), c.viewMonth.Month()+1, 0, 0, 0, 0, 0, c.viewMonth.Location()).Day()
	startWeekday := (int(firstDay.Weekday()) + 6) % 7

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	selDate := time.Date(c.cursorDate.Year(), c.cursorDate.Month(), c.cursorDate.Day(), 0, 0, 0, 0, c.cursorDate.Location())

	var rows []string
	row := ""
	for i := 0; i < startWeekday; i++ {
		row += calDayStyle.Render("")
	}
	for day := 1; day <= daysInMonth; day++ {
		d := time.Date(c.viewMonth.Year(), c.viewMonth.Month(), day, 0, 0, 0, 0, c.viewMonth.Location())
		label := fmt.Sprintf("%2d", day)

		var cell string
		if d.Equal(selDate) {
			cell = calSelectedStyle.Render(label)
		} else if d.Equal(today) {
			cell = calTodayStyle.Render(label)
		} else {
			cell = calDayStyle.Render(label)
		}
		row += cell

		if (startWeekday+day)%7 == 0 || day == daysInMonth {
			rows = append(rows, row)
			row = ""
		}
	}

	cal := lipgloss.JoinVertical(lipgloss.Top,
		header,
		weekdays,
		lipgloss.JoinVertical(lipgloss.Top, rows...),
	)

	return calBorderStyle.
		Width(calW).
		Height(calH).
		Render(cal)
}
