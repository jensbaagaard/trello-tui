package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jensbaagaard/trello-tui/internal/trello"
)

var timeNow = time.Now

func formatDue(due string, complete bool) (string, lipgloss.Style) {
	t, err := time.Parse(time.RFC3339Nano, due)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z", due)
		if err != nil {
			return "", lipgloss.NewStyle()
		}
	}

	label := t.Format("2 Jan")

	if complete {
		return "✓ " + label, dueDoneStyle
	}

	now := timeNow()
	if t.Before(now) {
		return "⚠ " + label, dueOverdueStyle
	}
	if t.Before(now.AddDate(0, 0, 7)) {
		return "⚠ " + label, dueSoonStyle
	}
	return label, dueDefaultStyle
}

func measureCard(c trello.Card, width int) int {
	rendered := renderCard(c, width, false)
	return lipgloss.Height(rendered)
}

func (m BoardModel) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.err)) +
			"\n\n" + helpStyle.Render("r:retry  esc:back to boards")
	}
	if m.loading {
		return fmt.Sprintf("Loading board: %s...", m.board.Name)
	}
	if len(m.lists) == 0 {
		return "No lists found on this board.\n\n" +
			helpStyle.Render("N:new list  r:refresh  esc:back")
	}

	if m.showHelp {
		return m.renderBoardHelp()
	}

	if m.mode == boardArchive || m.mode == boardArchiveFilter {
		return m.renderArchiveView()
	}

	if m.mode == boardLabelManager || m.mode == boardLabelCreate || m.mode == boardLabelEdit ||
		m.mode == boardLabelColorPick || m.mode == boardLabelConfirmDelete {
		return m.renderLabelManager()
	}

	if m.mode == boardMemberManager || m.mode == boardInviteMember || m.mode == boardConfirmRemoveMember {
		return m.renderMemberManager()
	}

	visLists := m.visibleLists()

	if m.filterText != "" && len(visLists) == 0 {
		header := titleStyle.Render(m.board.Name)
		noResults := helpStyle.Render(fmt.Sprintf("\nNo cards match \"%s\"", m.filterText))
		status := helpStyle.Render("esc:clear filter")
		return header + noResults + "\n\n" + status
	}

	colW := m.colWidth()
	vis := m.visibleColCount()
	start := m.colOffset
	end := start + vis
	if end > len(visLists) {
		end = len(visLists)
	}

	columns := make([]string, 0, vis)
	for i := start; i < end; i++ {
		columns = append(columns, m.renderColumn(i, visLists[i], colW))
	}

	board := lipgloss.JoinHorizontal(lipgloss.Top, columns...)

	var scrollHint string
	if start > 0 || end < len(visLists) {
		scrollHint = helpStyle.Render(fmt.Sprintf(" [%d-%d of %d lists]", start+1, end, len(visLists)))
	}

	var status string
	if m.mode == boardFilter {
		status = "Filter: " + m.textInput.View()
	} else if m.mode == boardAddCard {
		status = "New card: " + m.textInput.View()
	} else if m.mode == boardAddList {
		status = "New list: " + m.textInput.View()
	} else if m.mode == boardRenameList {
		status = "Rename list: " + m.textInput.View()
	} else if m.mode == boardConfirmArchive {
		card := m.selectedCard()
		name := ""
		if card != nil {
			name = card.Name
		}
		status = errorStyle.Render(fmt.Sprintf("Archive \"%s\"? (y/n)", name))
	} else if m.mode == boardConfirmArchiveList {
		vis := m.visibleLists()
		name := ""
		if m.activeList >= 0 && m.activeList < len(vis) {
			name = vis[m.activeList].Name
		}
		status = errorStyle.Render(fmt.Sprintf("Archive list \"%s\"? (y/n)", name))
	} else if m.statusMsg != "" {
		status = m.statusMsg
	} else if m.pendingAction != "" {
		status = helpStyle.Render(m.pendingAction)
	} else if m.filterText != "" {
		status = helpStyle.Render(fmt.Sprintf("filter: \"%s\"  esc:clear  /:edit  ?:help", m.filterText))
	} else {
		status = helpStyle.Render("←→:lists  j/k:cards  enter:open  n:new card  /:filter  ?:help")
	}

	var out strings.Builder
	out.WriteString(helpStyle.Render("Boards > ") + titleStyle.Render(m.board.Name))
	out.WriteString(scrollHint)
	out.WriteString("\n")
	out.WriteString(board)
	out.WriteString("\n")
	out.WriteString(status)
	return out.String()
}

func (m BoardModel) renderLabelManager() string {
	header := titleStyle.Render(m.board.Name+" — Labels") + "\n\n"
	var b strings.Builder

	switch m.mode {
	case boardLabelCreate, boardLabelEdit:
		action := "Create"
		if m.mode == boardLabelEdit {
			action = "Edit"
		}
		sT := sectionTitleStyle
		b.WriteString(sT.Render(action+" label") + "\n\n")
		b.WriteString("Name: " + m.labelNameInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("enter:pick color  esc:cancel"))

	case boardLabelColorPick:
		sT := sectionTitleStyle
		name := strings.TrimSpace(m.labelNameInput.Value())
		if name == "" {
			name = "(unnamed)"
		}
		b.WriteString(sT.Render("Pick color for: "+name) + "\n\n")
		for i, c := range TrelloColors {
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.labelColorIdx {
				cursor = "▸ "
				s = titleStyle
			}
			b.WriteString(cursor + labelColor(c).Render("● ") + s.Render(c) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:confirm  esc:back"))

	case boardLabelConfirmDelete:
		name := ""
		if m.labelCursor < len(m.boardLabels) {
			label := m.boardLabels[m.labelCursor]
			name = label.Name
			if name == "" {
				name = label.Color
			}
		}
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete label \"%s\"? This removes it from all cards. (y/n)", name)))

	default: // boardLabelManager
		if len(m.boardLabels) == 0 {
			b.WriteString(helpStyle.Render("No labels on this board."))
		} else {
			for i, label := range m.boardLabels {
				cursor := "  "
				s := lipgloss.NewStyle()
				if i == m.labelCursor {
					cursor = "▸ "
					s = titleStyle
				}
				name := label.Name
				if name == "" {
					name = "(unnamed)"
				}
				b.WriteString(cursor + labelColor(label.Color).Render("● ") + s.Render(name) + helpStyle.Render("  ["+label.Color+"]") + "\n")
			}
		}
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString(successMsgStyle.Render(m.statusMsg) + "\n")
		}
		b.WriteString(helpStyle.Render("j/k:navigate  n:new  e:edit  d:delete  esc:back"))
	}

	return header + b.String()
}

func (m BoardModel) renderMemberManager() string {
	header := titleStyle.Render(m.board.Name+" — Members") + "\n\n"
	var b strings.Builder

	switch m.mode {
	case boardInviteMember:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Invite member") + "\n\n")
		b.WriteString("Email: " + m.memberEmailInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("enter:invite  esc:cancel"))

	case boardConfirmRemoveMember:
		name := ""
		if m.memberCursor < len(m.boardMembers) {
			mem := m.boardMembers[m.memberCursor]
			name = mem.FullName
			if name == "" {
				name = mem.Username
			}
		}
		b.WriteString(errorStyle.Render(fmt.Sprintf("Remove \"%s\" from board? (y/n)", name)))

	default: // boardMemberManager
		if m.loadingMembers {
			b.WriteString(helpStyle.Render("Loading members..."))
		} else if len(m.boardMembers) == 0 {
			b.WriteString(helpStyle.Render("No members on this board."))
		} else {
			for i, mem := range m.boardMembers {
				cursor := "  "
				s := lipgloss.NewStyle()
				if i == m.memberCursor {
					cursor = "▸ "
					s = titleStyle
				}
				name := mem.FullName
				if name == "" {
					name = mem.Username
				}
				b.WriteString(cursor + s.Render(name) + helpStyle.Render("  @"+mem.Username) + "\n")
			}
		}
		b.WriteString("\n")
		if m.statusMsg != "" {
			b.WriteString(successMsgStyle.Render(m.statusMsg) + "\n")
		}
		b.WriteString(helpStyle.Render("j/k:navigate  n:invite  d:remove  esc:back"))
	}

	return header + b.String()
}

func (m BoardModel) renderArchiveView() string {
	header := titleStyle.Render(m.board.Name+" — Archived Cards") + "\n\n"
	var b strings.Builder

	cards := m.filteredArchivedCards()

	if len(m.archivedCards) == 0 {
		b.WriteString(helpStyle.Render("(no archived cards)"))
	} else if len(cards) == 0 {
		b.WriteString(helpStyle.Render(fmt.Sprintf("No cards match \"%s\"", m.archiveFilterText)))
	} else {
		visible := m.archiveVisibleCount()
		start := m.archiveScrollTop
		end := start + visible
		if end > len(cards) {
			end = len(cards)
		}

		if start > 0 {
			b.WriteString(dimRender(fmt.Sprintf("  ↑ %d more", start)) + "\n")
		}

		for i := start; i < end; i++ {
			card := cards[i]
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.archiveCursor {
				cursor = "▸ "
				s = titleStyle
			}
			listName := card.IDList
			for _, l := range m.lists {
				if l.ID == card.IDList {
					listName = l.Name
					break
				}
			}
			if listName == card.IDList {
				listName = "(deleted list)"
			}
			b.WriteString(cursor + s.Render(card.Name) + helpStyle.Render("  ["+listName+"]") + "\n")
		}

		if end < len(cards) {
			b.WriteString(dimRender(fmt.Sprintf("  ↓ %d more", len(cards)-end)) + "\n")
		}
	}

	b.WriteString("\n")
	if m.mode == boardArchiveFilter {
		b.WriteString("Filter: " + m.textInput.View() + "\n")
	} else if m.statusMsg != "" {
		b.WriteString(successMsgStyle.Render(m.statusMsg) + "\n")
	} else if m.archiveFilterText != "" {
		b.WriteString(helpStyle.Render(fmt.Sprintf("filter: %s", m.archiveFilterText)) + "\n")
	}
	if m.mode != boardArchiveFilter {
		b.WriteString(helpStyle.Render("j/k:navigate  enter/u:restore  /:filter  r:refresh  esc:back"))
	}

	return header + b.String()
}

func (m BoardModel) renderColumn(idx int, l trello.List, width int) string {
	isActive := idx == m.activeList
	allCards := m.cardsByList[l.ID]
	cards := m.filteredCards(l.ID)
	budget := m.cardBudget()
	colH := m.columnHeight()
	innerWidth := width - 2
	cardInner := m.cardInnerWidth()

	var titleText string
	if m.filterText != "" {
		titleText = fmt.Sprintf("%s (%d/%d)", l.Name, len(cards), len(allCards))
	} else {
		titleText = fmt.Sprintf("%s (%d)", l.Name, len(cards))
	}
	title := columnTitleStyle.Width(innerWidth).Render(titleText)

	scrollTop := 0
	if isActive {
		scrollTop = m.scrollTop
	}

	var cardViews []string
	used := 0

	if scrollTop > 0 {
		indicator := dimRender(fmt.Sprintf("  ↑ %d more", scrollTop))
		used += lipgloss.Height(indicator)
		cardViews = append(cardViews, indicator)
	}

	lastRendered := scrollTop
	for i := scrollTop; i < len(cards); i++ {
		selected := isActive && i == m.activeCard
		rendered := renderCard(cards[i], cardInner, selected)
		h := lipgloss.Height(rendered)

		remaining := budget - used
		needsDown := i < len(cards)-1
		reserve := 0
		if needsDown {
			reserve = 1
		}

		if used > 0 && h > remaining-reserve {
			break
		}
		cardViews = append(cardViews, rendered)
		used += h
		lastRendered = i + 1
	}

	if lastRendered < len(cards) {
		cardViews = append(cardViews, dimRender(fmt.Sprintf("  ↓ %d more", len(cards)-lastRendered)))
	}

	content := title + "\n" + strings.Join(cardViews, "\n")

	style := columnStyle
	if isActive {
		style = activeColumnStyle
	}

	return style.Width(innerWidth).Height(colH).Render(content)
}

func renderCard(c trello.Card, width int, selected bool) string {
	var b strings.Builder

	// Top line: labels + due date
	var parts []string
	if len(c.Labels) > 0 {
		var pills []string
		for _, l := range c.Labels {
			pills = append(pills, labelColor(l.Color).Render("━━"))
		}
		parts = append(parts, strings.Join(pills, " "))
	}
	if c.Due != "" {
		if label, s := formatDue(c.Due, c.DueComplete); label != "" {
			parts = append(parts, s.Render(label))
		}
	}
	if len(parts) > 0 {
		b.WriteString(strings.Join(parts, "  "))
		b.WriteString("\n")
	}

	b.WriteString(c.Name)

	// Bottom line: members + checklist badge
	var bottomParts []string
	if len(c.Members) > 0 {
		var memberBadges []string
		for _, m := range c.Members {
			memberBadges = append(memberBadges, memberColor(m.ID).Render(memberInitials(m.FullName)))
		}
		bottomParts = append(bottomParts, strings.Join(memberBadges, " "))
	}
	if c.Badges.CheckItems > 0 {
		checkBadge := fmt.Sprintf("☑ %d/%d", c.Badges.CheckItemsChecked, c.Badges.CheckItems)
		s := helpStyle
		if c.Badges.CheckItemsChecked == c.Badges.CheckItems {
			s = successMsgStyle
		}
		bottomParts = append(bottomParts, s.Render(checkBadge))
	}
	if len(bottomParts) > 0 {
		b.WriteString("\n\n")
		b.WriteString(strings.Join(bottomParts, "  "))
	}

	content := b.String()

	style := cardStyle
	if selected {
		style = selectedCardStyle
	} else if c.Due != "" && !c.DueComplete {
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", time.RFC3339} {
			if t, err := time.Parse(layout, c.Due); err == nil {
				now := timeNow()
				if t.Before(now) {
					style = cardStyle.BorderForeground(lipgloss.Color("#EF4444"))
				} else if t.Before(now.AddDate(0, 0, 7)) {
					style = cardStyle.BorderForeground(lipgloss.Color("#F59E0B"))
				}
				break
			}
		}
	}

	return style.Width(width).Render(content)
}

func memberInitials(name string) string {
	runes := []rune(strings.ToLower(strings.TrimSpace(name)))
	if len(runes) > 3 {
		runes = runes[:3]
	}
	return string(runes)
}

func truncate(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		if max > 3 {
			return string(runes[:max-3]) + "..."
		}
		return string(runes[:max])
	}
	return s
}

func dimRender(s string) string {
	return helpStyle.Render(s)
}

func (m BoardModel) renderBoardHelp() string {
	var sections []helpSection
	switch m.mode {
	case boardArchive:
		sections = []helpSection{
			{Title: "Navigation", Entries: []helpEntry{
				{"j/k", "Move up/down"},
				{"enter/u", "Restore card"},
				{"/", "Filter archived cards"},
				{"r", "Refresh"},
				{"esc", "Back (clear filter or exit)"},
			}},
		}
		return renderHelpOverlay("Archived Cards — Help", sections, m.width, m.height)
	case boardLabelManager:
		sections = []helpSection{
			{Title: "Labels", Entries: []helpEntry{
				{"j/k", "Move up/down"},
				{"n", "New label"},
				{"e", "Edit label"},
				{"d", "Delete label"},
				{"esc", "Back to board"},
			}},
		}
		return renderHelpOverlay("Label Manager — Help", sections, m.width, m.height)
	case boardMemberManager:
		sections = []helpSection{
			{Title: "Members", Entries: []helpEntry{
				{"j/k", "Move up/down"},
				{"n", "Invite member"},
				{"d", "Remove member"},
				{"esc", "Back to board"},
			}},
		}
		return renderHelpOverlay("Member Manager — Help", sections, m.width, m.height)
	}

	sections = []helpSection{
		{Title: "Navigation", Entries: []helpEntry{
			{"left/right", "Switch lists"},
			{"j/k", "Move up/down cards"},
			{"enter", "Open card"},
			{"?", "Toggle help"},
			{"esc", "Back (clear filter or exit)"},
		}},
		{Title: "Cards", Entries: []helpEntry{
			{"n", "New card"},
			{"c", "Archive card"},
			{",/.", "Move card left/right"},
			{"</>", "Move card to first/last list"},
		}},
		{Title: "Lists", Entries: []helpEntry{
			{"N", "New list"},
			{"R", "Rename list"},
			{"C", "Archive list"},
			{"{/}", "Move list left/right"},
		}},
		{Title: "Tools", Entries: []helpEntry{
			{"/", "Filter cards"},
			{"L", "Label manager"},
			{"M", "Member manager"},
			{"a", "View archived cards"},
			{"r", "Refresh board"},
		}},
	}
	return renderHelpOverlay("Board — Help", sections, m.width, m.height)
}
