package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m CardModel) View() string {
	if m.showHelp {
		return m.renderCardHelp()
	}

	w := m.width - 2
	if w < 44 {
		w = 44
	}

	available := m.height - 3
	if available < 24 {
		available = 24
	}

	var statusBar string
	if m.statusMsg != "" {
		statusBar = successMsgStyle.Render(m.statusMsg)
	} else {
		statusBar = m.helpLine()
	}

	showChecklist := m.loadingCL || len(m.checklists) > 0 || m.mode == cardAddChecklist || m.mode == cardAddCheckItem || m.mode == cardConfirmDeleteChecklist
	showAttachments := len(m.attachments) > 0 || m.mode == cardAddAttachment || m.mode == cardConfirmDeleteAttachment

	paneCount := 2 // info + activity always
	if showChecklist {
		paneCount++
	}
	if showAttachments {
		paneCount++
	}

	// overhead: paneCount newlines between panes + status bar lines
	statusLines := strings.Count(statusBar, "\n") + 1
	overhead := paneCount + statusLines
	usable := available - overhead

	var pane1H, pane2H, paneAttH, paneActH int
	switch {
	case showChecklist && showAttachments:
		pane1H = usable * infoPanePercent4 / 100
		pane2H = usable * checklistPanePercent / 100
		paneAttH = usable * attachPanePercent4 / 100
		paneActH = usable - pane1H - pane2H - paneAttH
	case showChecklist:
		pane1H = usable * infoPanePercent3CL / 100
		pane2H = usable * checklistPanePercent3 / 100
		paneActH = usable - pane1H - pane2H
	case showAttachments:
		pane1H = usable * infoPanePercent3Att / 100
		paneAttH = usable * attachPanePercent3 / 100
		paneActH = usable - pane1H - paneAttH
	default:
		pane1H = usable * infoPanePercent2 / 100
		paneActH = usable - pane1H
	}
	if pane1H < minInfoPaneHeight {
		pane1H = minInfoPaneHeight
	}
	if pane2H > 0 && pane2H < minSubPaneHeight {
		pane2H = minSubPaneHeight
	}
	if paneAttH > 0 && paneAttH < minSubPaneHeight {
		paneAttH = minSubPaneHeight
	}
	if paneActH < minSubPaneHeight {
		paneActH = minSubPaneHeight
	}

	box1 := m.renderInfoPane(w, pane1H, m.infoScroll)
	body := box1
	if showChecklist {
		body += "\n" + m.renderChecklistPane(w, pane2H, m.checkItemIdx)
	}
	if showAttachments {
		body += "\n" + m.renderAttachmentsPane(w, paneAttH, m.attachmentIdx)
	}
	body += "\n" + m.renderActivityPane(w, paneActH, m.activityIdx) + "\n" + statusBar
	return lipgloss.NewStyle().Padding(0, 1).Render(body)
}

func (m CardModel) renderInfoPane(width, height, scroll int) string {
	active := m.mode != cardChecklistPane && m.mode != cardAttachmentsPane && m.mode != cardActivityPane && m.mode != cardAddComment && m.mode != cardAddChecklist && m.mode != cardAddCheckItem && m.mode != cardAddAttachment && m.mode != cardConfirmDeleteChecklist && m.mode != cardConfirmDeleteAttachment

	var b strings.Builder

	switch m.mode {
	case cardMoveBoard:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Move to board") + "\n")
		b.WriteString(m.pickerFilter.View() + "\n\n")
		if len(m.allBoards) == 0 {
			b.WriteString(helpStyle.Render("Loading..."))
		} else {
			filtered := m.filteredBoards()
			if len(filtered) == 0 {
				b.WriteString(helpStyle.Render("No matches"))
			} else {
				for i, board := range filtered {
					cursor := "  "
					s := lipgloss.NewStyle()
					if i == m.boardIndex {
						cursor = "▸ "
						s = titleStyle
					}
					b.WriteString(cursor + s.Render(board.Name) + "\n")
				}
			}
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:select  esc:back"))

	case cardMoveBoardList:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Move to list on "+m.targetBoard.Name) + "\n\n")
		for i, l := range m.targetLists {
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.targetListIndex {
				cursor = "▸ "
				s = titleStyle
			}
			b.WriteString(cursor + s.Render(l.Name) + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:move  esc:back"))

	case cardEditTitle:
		b.WriteString("Edit title (enter:save  esc:cancel):\n\n")
		b.WriteString(m.titleEdit.View())

	case cardEditDesc:
		b.WriteString("Description (esc:save):\n\n")
		b.WriteString(m.descEdit.View())

	case cardMoveList:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Move to list") + "\n\n")
		for i, l := range m.lists {
			cursor := "  "
			s := lipgloss.NewStyle()
			if i == m.moveIndex {
				cursor = "▸ "
				s = titleStyle
			}
			suffix := ""
			if i == m.listIndex {
				suffix = helpStyle.Render("  (current)")
			}
			b.WriteString(cursor + s.Render(l.Name) + suffix + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:move  B:other board  esc:cancel"))

	case cardAddMember:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Add / remove member") + "\n")
		b.WriteString(m.pickerFilter.View() + "\n\n")
		if len(m.boardMembers) == 0 {
			b.WriteString(helpStyle.Render("Loading..."))
		} else {
			filtered := m.filteredMembers()
			if len(filtered) == 0 {
				b.WriteString(helpStyle.Render("No matches"))
			} else {
				for i, member := range filtered {
					cursor := "  "
					s := lipgloss.NewStyle()
					if i == m.memberIndex {
						cursor = "▸ "
						s = titleStyle
					}
					check := "  "
					if m.isOnCard(member.ID) {
						check = successMsgStyle.Render("✓ ")
					}
					b.WriteString(cursor + check + s.Render(member.FullName) + "\n")
				}
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space:toggle  esc:close"))

	case cardAddLabel:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Add / remove label") + "\n")
		b.WriteString(m.pickerFilter.View() + "\n\n")
		if len(m.boardLabels) == 0 {
			b.WriteString(helpStyle.Render("Loading..."))
		} else {
			filtered := m.filteredLabels()
			if len(filtered) == 0 {
				b.WriteString(helpStyle.Render("No matches"))
			} else {
				for i, label := range filtered {
					cursor := "  "
					rs := lipgloss.NewStyle()
					if i == m.labelIndex {
						cursor = "▸ "
						rs = titleStyle
					}
					check := "  "
					if m.isLabelOnCard(label.ID) {
						check = successMsgStyle.Render("✓ ")
					}
					name := label.Name
					if name == "" {
						name = label.Color
					}
					b.WriteString(cursor + check + labelColor(label.Color).Render("● ") + rs.Render(name) + "\n")
				}
			}
		}
		b.WriteString("\n" + helpStyle.Render("enter/space:toggle  ctrl+n:new label  esc:close"))

	case cardCreateLabel:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Create label") + "\n\n")
		b.WriteString("Name: " + m.labelNameInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("enter:pick color  esc:cancel"))

	case cardCreateLabelColor:
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
		b.WriteString("\n" + helpStyle.Render("j/k:navigate  enter:create  esc:back"))

	case cardSetDue:
		sT := sectionTitleStyle
		b.WriteString(sT.Render("Set due date") + "\n\n")
		b.WriteString("Date (YYYY-MM-DD, empty to clear):\n\n")
		b.WriteString(m.dueInput.View())
		b.WriteString("\n\n" + helpStyle.Render("enter:save  esc:cancel"))

	default:
		// breadcrumb
		crumb := m.listName
		if m.boardName != "" {
			crumb = m.boardName + " > " + crumb
		}
		b.WriteString(helpStyle.Render(crumb) + "\n\n")

		// title
		b.WriteString(boldWhiteStyle.Render(m.card.Name) + "\n")

		// labels
		if len(m.card.Labels) > 0 {
			var parts []string
			for _, l := range m.card.Labels {
				name := l.Name
				if name == "" {
					name = l.Color
				}
				parts = append(parts, labelColor(l.Color).Render(name))
			}
			b.WriteString("\n" + strings.Join(parts, "  ") + "\n")
		}

		// due date
		if m.card.Due != "" {
			if label, s := formatDue(m.card.Due, m.card.DueComplete); label != "" {
				b.WriteString("\n" + s.Render("Due: "+label) + "\n")
			}
		}

		// members
		if len(m.card.Members) > 0 {
			var names []string
			for _, mem := range m.card.Members {
				names = append(names, mem.FullName)
			}
			b.WriteString("\n" + helpStyle.Render("Members: "+strings.Join(names, ", ")) + "\n")
		}

		// url
		if m.card.ShortURL != "" {
			b.WriteString("\n" + helpStyle.Render(m.card.ShortURL) + "\n")
		}

		b.WriteString("\n")

		// description
		desc := m.card.Desc
		if desc == "" {
			desc = helpStyle.Render("(no description)")
		}
		b.WriteString(desc)
	}

	return paneBox("Card Info", b.String(), width, height, active, scroll)
}

func (m CardModel) renderChecklistPane(width, height, cursorIdx int) string {
	active := m.mode == cardChecklistPane || m.mode == cardAddChecklist || m.mode == cardAddCheckItem || m.mode == cardConfirmDeleteChecklist
	var b strings.Builder

	if m.mode == cardConfirmDeleteChecklist {
		refs := m.allCheckItemRefs()
		clIdx := 0
		if len(refs) > 0 && m.checkItemIdx < len(refs) {
			clIdx = refs[m.checkItemIdx].cl
		}
		name := m.checklists[clIdx].Name
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete checklist \"%s\"? (y/n)", name)))
	} else if m.mode == cardAddChecklist {
		b.WriteString(helpStyle.Render("New checklist (enter:create  esc:cancel)") + "\n\n")
		b.WriteString(m.checklistInput.View())
	} else if m.mode == cardAddCheckItem {
		b.WriteString(helpStyle.Render("New item (enter:add  esc:cancel)") + "\n\n")
		b.WriteString(m.checklistInput.View())
	} else if m.loadingCL {
		b.WriteString(helpStyle.Render("Loading..."))
	} else if len(m.checklists) == 0 {
		b.WriteString(helpStyle.Render("(no checklists)"))
	} else {
		refs := m.allCheckItemRefs()
		flatIdx := 0
		for _, cl := range m.checklists {
			// checklist header with progress
			done := 0
			for _, it := range cl.CheckItems {
				if it.State == "complete" {
					done++
				}
			}
			total := len(cl.CheckItems)
			clTitle := lipgloss.NewStyle().Bold(true).Render(
				fmt.Sprintf("%s (%d/%d)", cl.Name, done, total),
			)
			b.WriteString(clTitle + "\n")

			for _, item := range cl.CheckItems {
				cursor := "  "
				itemStyle := lipgloss.NewStyle()
				if m.mode == cardChecklistPane && flatIdx < len(refs) && m.checkItemIdx == flatIdx {
					cursor = "▸ "
					itemStyle = titleStyle
				}
				box := "[ ]"
				if item.State == "complete" {
					box = successMsgStyle.Render("[x]")
					itemStyle = itemStyle.Foreground(dimColor)
				}
				b.WriteString(cursor + box + " " + itemStyle.Render(item.Name) + "\n")
				flatIdx++
			}
		}
	}

	title := "Checklist"
	if m.mode == cardChecklistPane {
		title += helpStyle.Render("  j/k:navigate  enter:toggle  n:add item  -:new checklist  d:delete checklist  tab:next  esc:back")
	}
	innerW := width - 4
	if innerW < 10 {
		innerW = 10
	}
	availLines := (height - 2) - 2
	if availLines < 1 {
		availLines = 1
	}
	scroll := clampScroll(cursorIdx, availLines)
	return paneBox(title, b.String(), width, height, active, scroll)
}

func (m CardModel) renderAttachmentsPane(width, height, cursorIdx int) string {
	active := m.mode == cardAttachmentsPane || m.mode == cardAddAttachment || m.mode == cardConfirmDeleteAttachment
	var b strings.Builder

	if m.mode == cardConfirmDeleteAttachment {
		name := m.attachments[m.attachmentIdx].Name
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete attachment \"%s\"? (y/n)", name)))
	} else if m.mode == cardAddAttachment {
		b.WriteString(helpStyle.Render("Add URL attachment (enter:add  esc:cancel)") + "\n\n")
		b.WriteString(m.checklistInput.View())
	} else if m.loadingAtt {
		b.WriteString(helpStyle.Render("Loading..."))
	} else if len(m.attachments) == 0 {
		b.WriteString(helpStyle.Render("(no attachments)"))
	} else {
		for i, att := range m.attachments {
			cursor := "  "
			s := lipgloss.NewStyle()
			if m.mode == cardAttachmentsPane && i == m.attachmentIdx {
				cursor = "▸ "
				s = titleStyle
			}
			size := formatBytes(att.Bytes)
			b.WriteString(cursor + s.Render(att.Name) + helpStyle.Render("  "+size) + "\n")
		}
	}

	title := "Attachments"
	if m.mode == cardAttachmentsPane {
		title += helpStyle.Render("  j/k:navigate  o:open  a:add URL  d:delete  tab:next  esc:back")
	}
	availLines := (height - 2) - 2
	if availLines < 1 {
		availLines = 1
	}
	scroll := clampScroll(cursorIdx, availLines)
	return paneBox(title, b.String(), width, height, active, scroll)
}

func formatBytes(b int) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func (m CardModel) renderActivityPane(width, height, cursorIdx int) string {
	active := m.mode == cardActivityPane || m.mode == cardAddComment
	var b strings.Builder

	if m.mode == cardAddComment {
		b.WriteString(helpStyle.Render("New comment (ctrl+s:send  esc:cancel)") + "\n\n")
		b.WriteString(m.commentInput.View())
	} else if m.loadingCom {
		b.WriteString(helpStyle.Render("Loading..."))
	} else if len(m.actions) == 0 {
		b.WriteString(helpStyle.Render("(no activity)"))
	} else {
		for i, a := range m.actions {
			cursor := "  "
			if active && i == m.activityIdx {
				cursor = "▸ "
			}

			dateStr := ""
			for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
				if t, err := time.Parse(layout, a.Date); err == nil {
					dateStr = t.Format("2 Jan")
					break
				}
			}

			author := a.MemberCreator.FullName
			if author == "" {
				author = a.MemberCreator.Username
			}

			switch a.Type {
			case "commentCard":
				header := lipgloss.NewStyle().Bold(true).Render(author)
				if dateStr != "" {
					header += helpStyle.Render(" • " + dateStr)
				}
				b.WriteString(cursor + header + "\n")
				b.WriteString("  " + a.Data.Text + "\n\n")

			case "updateCard":
				var line string
				if a.Data.ListBefore != nil && a.Data.ListAfter != nil {
					line = fmt.Sprintf("%s moved this card from %s to %s", author, a.Data.ListBefore.Name, a.Data.ListAfter.Name)
				} else {
					line = fmt.Sprintf("%s updated this card", author)
				}
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "createCard":
				listName := ""
				if a.Data.List != nil {
					listName = a.Data.List.Name
				}
				line := fmt.Sprintf("%s added this card to %s", author, listName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "addMemberToCard":
				memberName := ""
				if a.Data.Member != nil {
					memberName = a.Data.Member.FullName
				}
				line := fmt.Sprintf("%s added %s to this card", author, memberName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "removeMemberFromCard":
				memberName := ""
				if a.Data.Member != nil {
					memberName = a.Data.Member.FullName
				}
				line := fmt.Sprintf("%s removed %s from this card", author, memberName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			case "addAttachmentToCard":
				attachName := ""
				if a.Data.Attachment != nil {
					attachName = a.Data.Attachment.Name
				}
				line := fmt.Sprintf("%s attached %s", author, attachName)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")

			default:
				line := fmt.Sprintf("%s performed an action", author)
				if dateStr != "" {
					line += " • " + dateStr
				}
				b.WriteString(cursor + helpStyle.Render(line) + "\n\n")
			}
		}
	}

	title := "Activity"
	if active && m.mode != cardAddComment {
		title += helpStyle.Render("  j/k:scroll  n:comment  tab:next  esc:back")
	}
	availLines := (height - 2) - 2
	if availLines < 1 {
		availLines = 1
	}
	// each activity item is ~2-3 lines; scroll to keep cursor visible
	scroll := clampScroll(cursorIdx*3, availLines)
	return paneBox(title, b.String(), width, height, active, scroll)
}

// clampScroll returns a scroll offset that ensures the cursor line is visible.
func clampScroll(cursorLine, visibleLines int) int {
	if cursorLine < visibleLines {
		return 0
	}
	return cursorLine - visibleLines + 1
}

func (m CardModel) helpLine() string {
	switch m.mode {
	case cardChecklistPane, cardAddChecklist, cardAddCheckItem, cardConfirmDeleteChecklist:
		return ""
	case cardAttachmentsPane, cardAddAttachment, cardConfirmDeleteAttachment:
		return ""
	case cardActivityPane:
		return ""
	case cardAddComment:
		return ""
	case cardMoveBoard, cardMoveBoardList:
		return ""
	default:
		if m.pendingAction != "" {
			return helpStyle.Render(m.pendingAction)
		}
		return wrapHelpText("t:title  e:desc  m:move  tab:pane  ?:help  esc:back", m.width-2)
	}
}

func wrapHelpText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return helpStyle.Render(text)
	}
	items := strings.Split(text, "  ")
	var lines []string
	cur := ""
	for _, item := range items {
		if cur == "" {
			cur = item
		} else if len(cur)+2+len(item) <= maxWidth {
			cur += "  " + item
		} else {
			lines = append(lines, cur)
			cur = item
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	var rendered []string
	for _, l := range lines {
		rendered = append(rendered, helpStyle.Render(l))
	}
	return strings.Join(rendered, "\n")
}

func (m CardModel) renderCardHelp() string {
	sections := []helpSection{
		{Title: "Info Pane", Entries: []helpEntry{
			{"t", "Edit title"},
			{"e", "Edit description"},
			{"m", "Move to list"},
			{"B", "Move to another board"},
			{"a", "Add/remove members"},
			{"l", "Add/remove labels"},
			{"ctrl+n", "Create new label (from picker)"},
			{"d", "Set due date"},
			{"c", "Copy card URL"},
			{",/.", "Move card left/right"},
			{"</>", "Move card to first/last list"},
			{"-", "New checklist"},
			{"A", "Attach URL"},
		}},
		{Title: "Navigation", Entries: []helpEntry{
			{"tab", "Next pane"},
			{"j/k", "Scroll"},
			{"?", "Toggle help"},
			{"esc", "Back"},
		}},
		{Title: "Checklist Pane", Entries: []helpEntry{
			{"enter", "Toggle check item"},
			{"n", "Add item"},
			{"-", "New checklist"},
			{"d", "Delete checklist"},
		}},
		{Title: "Attachments Pane", Entries: []helpEntry{
			{"o", "Open attachment"},
			{"a", "Add URL"},
			{"d", "Delete attachment"},
		}},
		{Title: "Activity Pane", Entries: []helpEntry{
			{"n", "New comment (ctrl+s to send)"},
		}},
	}
	return renderHelpOverlay("Card — Help", sections, m.width, m.height)
}

// ── paneBox ───────────────────────────────────────────────────────────────────

func paneBox(title, content string, width, height int, active bool, scroll int) string {
	borderColor := dimColor
	if active {
		borderColor = primaryColor
	}
	titleColor := secondaryColor
	if !active {
		titleColor = dimColor
	}
	innerW := width - 4 // border(1 each side) + padding(1 each side)
	if innerW < 10 {
		innerW = 10
	}
	innerH := height - 2 // subtract top+bottom borders
	titleLine := lipgloss.NewStyle().Bold(true).Foreground(titleColor).Render(title)
	titleLines := lipgloss.Height(titleLine)
	if titleW := lipgloss.Width(titleLine); titleW > innerW && innerW > 0 {
		titleLines = (titleW + innerW - 1) / innerW
	}
	// title lines + 1 blank line
	availLines := innerH - titleLines - 1
	if availLines < 1 {
		availLines = 1
	}
	content = scrollLines(content, scroll, availLines, innerW)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(innerW).
		Height(innerH).
		Render(titleLine + "\n\n" + content)
}

// scrollLines skips the first `offset` visual rows then returns up to maxLines rows,
// accounting for long lines that wrap at wrapWidth characters.
func scrollLines(content string, offset, maxLines, wrapWidth int) string {
	lines := strings.Split(content, "\n")
	// expand each logical line into visual rows
	type row struct{ text string }
	var rows []row
	for _, line := range lines {
		vw := lipgloss.Width(line)
		if wrapWidth > 0 && vw > wrapWidth {
			// approximate: split into chunks by rune count
			runes := []rune(line)
			for len(runes) > 0 {
				end := wrapWidth
				if end > len(runes) {
					end = len(runes)
				}
				rows = append(rows, row{string(runes[:end])})
				runes = runes[end:]
			}
		} else {
			rows = append(rows, row{line})
		}
	}
	if offset >= len(rows) {
		offset = len(rows) - 1
	}
	if offset < 0 {
		offset = 0
	}
	visible := rows[offset:]
	if len(visible) > maxLines {
		visible = visible[:maxLines]
	}
	out := make([]string, len(visible))
	for i, r := range visible {
		out[i] = r.text
	}
	return strings.Join(out, "\n")
}
