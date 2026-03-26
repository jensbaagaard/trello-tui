package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m CardModel) handleKey(msg tea.KeyMsg) (CardModel, tea.Cmd) {
	if m.showHelp {
		if msg.String() == "?" || msg.String() == "esc" {
			m.showHelp = false
		}
		return m, nil
	}

	switch m.mode {
	case cardEditTitle:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.titleEdit.Value())
			if name != "" {
				m.card.Name = name
				m.mode = cardView
				m.pendingAction = "Saving title..."
				return m, m.updateCard(map[string]string{"name": name})
			}
			m.mode = cardView
		case "esc":
			m.mode = cardView
		default:
			var cmd tea.Cmd
			m.titleEdit, cmd = m.titleEdit.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardEditDesc:
		switch msg.String() {
		case "esc":
			desc := m.descEdit.Value()
			m.card.Desc = desc
			m.mode = cardView
			m.pendingAction = "Saving description..."
			return m, m.updateCard(map[string]string{"desc": desc})
		default:
			var cmd tea.Cmd
			m.descEdit, cmd = m.descEdit.Update(msg)
			return m, cmd
		}

	case cardMoveList:
		switch msg.String() {
		case "j", "down":
			if m.moveIndex < len(m.lists)-1 {
				m.moveIndex++
			}
		case "k", "up":
			if m.moveIndex > 0 {
				m.moveIndex--
			}
		case "enter":
			if m.moveIndex != m.listIndex {
				m.pendingAction = "Moving card..."
				return m, m.moveToList(m.moveIndex)
			}
			m.mode = cardView
		case "B":
			m.mode = cardMoveBoard
			m.boardIndex = 0
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			if len(m.allBoards) == 0 {
				return m, tea.Batch(textinput.Blink, m.fetchAllBoards())
			}
			return m, textinput.Blink
		case "esc":
			m.moveIndex = m.listIndex
			m.mode = cardView
		}
		return m, nil

	case cardMoveBoard:
		switch msg.String() {
		case "j", "down":
			filtered := m.filteredBoards()
			if m.boardIndex < len(filtered)-1 {
				m.boardIndex++
			}
		case "k", "up":
			if m.boardIndex > 0 {
				m.boardIndex--
			}
		case "enter":
			filtered := m.filteredBoards()
			if len(filtered) > 0 && m.boardIndex < len(filtered) {
				m.targetBoard = filtered[m.boardIndex]
				m.statusMsg = "Loading lists..."
				return m, m.fetchTargetBoardLists()
			}
		case "esc":
			m.mode = cardMoveList
			m.pickerFilter.SetValue("")
		default:
			prev := m.pickerFilter.Value()
			var cmd tea.Cmd
			m.pickerFilter, cmd = m.pickerFilter.Update(msg)
			if m.pickerFilter.Value() != prev {
				m.boardIndex = 0
			}
			return m, cmd
		}
		return m, nil

	case cardMoveBoardList:
		switch msg.String() {
		case "j", "down":
			if m.targetListIndex < len(m.targetLists)-1 {
				m.targetListIndex++
			}
		case "k", "up":
			if m.targetListIndex > 0 {
				m.targetListIndex--
			}
		case "enter":
			if len(m.targetLists) > 0 && m.targetListIndex < len(m.targetLists) {
				targetList := m.targetLists[m.targetListIndex]
				return m, m.moveToBoard(m.targetBoard.ID, targetList.ID)
			}
		case "esc":
			m.mode = cardMoveBoard
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case cardAddMember:
		switch msg.String() {
		case "j", "down":
			if m.memberIndex < len(m.filteredMembers())-1 {
				m.memberIndex++
			}
		case "k", "up":
			if m.memberIndex > 0 {
				m.memberIndex--
			}
		case "enter", " ":
			filtered := m.filteredMembers()
			if len(filtered) > 0 && m.memberIndex < len(filtered) {
				member := filtered[m.memberIndex]
				client := m.client
				cardID := m.card.ID
				memberID := member.ID
				for i, cm := range m.card.Members {
					if cm.ID == memberID {
						m.card.Members = append(m.card.Members[:i], m.card.Members[i+1:]...)
						m.pendingAction = "Updating member..."
						return m, func() tea.Msg {
							return MemberToggledMsg{Err: client.RemoveMemberFromCard(cardID, memberID)}
						}
					}
				}
				m.card.Members = append(m.card.Members, member)
				m.pendingAction = "Updating member..."
				return m, func() tea.Msg {
					return MemberToggledMsg{Err: client.AddMemberToCard(cardID, memberID)}
				}
			}
		case "esc":
			m.mode = cardView
			m.pickerFilter.SetValue("")
		default:
			prev := m.pickerFilter.Value()
			var cmd tea.Cmd
			m.pickerFilter, cmd = m.pickerFilter.Update(msg)
			if m.pickerFilter.Value() != prev {
				m.memberIndex = 0
			}
			return m, cmd
		}
		return m, nil

	case cardAddLabel:
		switch msg.String() {
		case "j", "down":
			if m.labelIndex < len(m.filteredLabels())-1 {
				m.labelIndex++
			}
		case "k", "up":
			if m.labelIndex > 0 {
				m.labelIndex--
			}
		case "enter", " ":
			filtered := m.filteredLabels()
			if len(filtered) > 0 && m.labelIndex < len(filtered) {
				label := filtered[m.labelIndex]
				client := m.client
				cardID := m.card.ID
				labelID := label.ID
				for i, cl := range m.card.Labels {
					if cl.ID == labelID {
						m.card.Labels = append(m.card.Labels[:i], m.card.Labels[i+1:]...)
						m.pendingAction = "Updating label..."
						return m, func() tea.Msg {
							return LabelToggledMsg{Err: client.RemoveLabelFromCard(cardID, labelID)}
						}
					}
				}
				m.card.Labels = append(m.card.Labels, label)
				m.pendingAction = "Updating label..."
				return m, func() tea.Msg {
					return LabelToggledMsg{Err: client.AddLabelToCard(cardID, labelID)}
				}
			}
		case "ctrl+n":
			m.mode = cardCreateLabel
			m.labelNameInput.SetValue("")
			m.labelNameInput.Focus()
			m.labelColorIdx = 0
			return m, textinput.Blink
		case "esc":
			m.mode = cardView
			m.pickerFilter.SetValue("")
		default:
			prev := m.pickerFilter.Value()
			var cmd tea.Cmd
			m.pickerFilter, cmd = m.pickerFilter.Update(msg)
			if m.pickerFilter.Value() != prev {
				m.labelIndex = 0
			}
			return m, cmd
		}
		return m, nil

	case cardCreateLabel:
		switch msg.String() {
		case "enter":
			m.mode = cardCreateLabelColor
			m.labelNameInput.Blur()
			return m, nil
		case "esc":
			m.mode = cardAddLabel
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			return m, textinput.Blink
		default:
			var cmd tea.Cmd
			m.labelNameInput, cmd = m.labelNameInput.Update(msg)
			return m, cmd
		}

	case cardCreateLabelColor:
		switch msg.String() {
		case "j", "down":
			if m.labelColorIdx < len(TrelloColors)-1 {
				m.labelColorIdx++
			}
		case "k", "up":
			if m.labelColorIdx > 0 {
				m.labelColorIdx--
			}
		case "enter":
			name := strings.TrimSpace(m.labelNameInput.Value())
			color := TrelloColors[m.labelColorIdx]
			client := m.client
			boardID := m.boardID
			return m, func() tea.Msg {
				label, err := client.CreateLabel(boardID, name, color)
				return LabelCreatedMsg{Label: label, Err: err}
			}
		case "esc":
			m.mode = cardCreateLabel
			m.labelNameInput.Focus()
			return m, textinput.Blink
		}
		return m, nil

	case cardSetDue:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.dueInput.Value())
			m.mode = cardView
			if val == "" {
				m.card.Due = ""
				m.card.DueComplete = false
				m.pendingAction = "Saving due date..."
				return m, m.updateCard(map[string]string{"due": ""})
			}
			t, err := time.Parse("2006-01-02", val)
			if err != nil {
				m.statusMsg = "Invalid date (use YYYY-MM-DD)"
				return m, nil
			}
			dueStr := t.UTC().Format(time.RFC3339)
			m.card.Due = dueStr
			m.pendingAction = "Saving due date..."
			return m, m.updateCard(map[string]string{"due": dueStr})
		case "esc":
			m.mode = cardView
		default:
			var cmd tea.Cmd
			m.dueInput, cmd = m.dueInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardChecklistPane:
		refs := m.allCheckItemRefs()
		switch msg.String() {
		case "?":
			m.showHelp = true
			return m, nil
		case "j", "down":
			if m.checkItemIdx < len(refs)-1 {
				m.checkItemIdx++
			}
		case "k", "up":
			if m.checkItemIdx > 0 {
				m.checkItemIdx--
			}
		case "enter", " ":
			if len(refs) > 0 && m.checkItemIdx < len(refs) {
				r := refs[m.checkItemIdx]
				item := m.checklists[r.cl].CheckItems[r.it]
				newComplete := item.State != "complete"
				if newComplete {
					m.checklists[r.cl].CheckItems[r.it].State = "complete"
				} else {
					m.checklists[r.cl].CheckItems[r.it].State = "incomplete"
				}
				client := m.client
				cardID := m.card.ID
				checkItemID := item.ID
				m.pendingAction = "Toggling item..."
				return m, func() tea.Msg {
					return CheckItemToggledMsg{Err: client.ToggleCheckItem(cardID, checkItemID, newComplete)}
				}
			}
		case "d":
			if len(m.checklists) > 0 {
				m.mode = cardConfirmDeleteChecklist
			}
		case "n":
			if len(m.checklists) > 0 {
				m.mode = cardAddCheckItem
				m.checklistInput.Placeholder = "Item name..."
				m.checklistInput.SetValue("")
				m.checklistInput.Focus()
				return m, textinput.Blink
			}
		case "-", "N":
			m.mode = cardAddChecklist
			m.checklistInput.Placeholder = "Checklist name..."
			m.checklistInput.SetValue("")
			m.checklistInput.Focus()
			return m, textinput.Blink
		case "tab":
			m.clScroll = 0
			if len(m.attachments) > 0 {
				m.mode = cardAttachmentsPane
			} else {
				m.mode = cardActivityPane
			}
		case "esc":
			m.clScroll = 0
			m.mode = cardView
		}
		return m, nil

	case cardConfirmDeleteChecklist:
		switch msg.String() {
		case "y", "Y":
			refs := m.allCheckItemRefs()
			clIdx := 0
			if len(refs) > 0 && m.checkItemIdx < len(refs) {
				clIdx = refs[m.checkItemIdx].cl
			}
			checklistID := m.checklists[clIdx].ID
			m.mode = cardChecklistPane
			return m, m.deleteChecklist(checklistID)
		case "n", "N", "esc":
			m.mode = cardChecklistPane
		}
		return m, nil

	case cardAttachmentsPane:
		switch msg.String() {
		case "?":
			m.showHelp = true
			return m, nil
		case "j", "down":
			if m.attachmentIdx < len(m.attachments)-1 {
				m.attachmentIdx++
			}
		case "k", "up":
			if m.attachmentIdx > 0 {
				m.attachmentIdx--
			}
		case "o", "enter":
			if len(m.attachments) > 0 && m.attachmentIdx < len(m.attachments) {
				return m, m.openAttachment(m.attachments[m.attachmentIdx])
			}
		case "d":
			if len(m.attachments) > 0 {
				m.mode = cardConfirmDeleteAttachment
			}
		case "a":
			m.mode = cardAddAttachment
			m.checklistInput.Placeholder = "URL..."
			m.checklistInput.SetValue("")
			m.checklistInput.Focus()
			return m, textinput.Blink
		case "tab":
			m.attScroll = 0
			m.mode = cardActivityPane
		case "esc":
			m.attScroll = 0
			m.mode = cardView
		}
		return m, nil

	case cardConfirmDeleteAttachment:
		switch msg.String() {
		case "y", "Y":
			att := m.attachments[m.attachmentIdx]
			m.mode = cardAttachmentsPane
			return m, m.deleteAttachment(att.ID)
		case "n", "N", "esc":
			m.mode = cardAttachmentsPane
		}
		return m, nil

	case cardActivityPane:
		switch msg.String() {
		case "?":
			m.showHelp = true
			return m, nil
		case "j", "down":
			if m.activityIdx < len(m.actions)-1 {
				m.activityIdx++
			}
		case "k", "up":
			if m.activityIdx > 0 {
				m.activityIdx--
			}
		case "n":
			m.mode = cardAddComment
			m.commentInput.SetValue("")
			m.commentInput.Focus()
			return m, textarea.Blink
		case "tab":
			m.actScroll = 0
			m.mode = cardView
		case "esc":
			m.actScroll = 0
			m.mode = cardView
		}
		return m, nil

	case cardAddComment:
		switch msg.String() {
		case "ctrl+s":
			text := strings.TrimSpace(m.commentInput.Value())
			if text == "" {
				m.mode = cardActivityPane
				return m, nil
			}
			m.mode = cardActivityPane
			m.pendingAction = "Adding comment..."
			client := m.client
			cardID := m.card.ID
			return m, func() tea.Msg {
				action, err := client.AddComment(cardID, text)
				return CommentAddedMsg{Action: action, Err: err}
			}
		case "esc":
			m.mode = cardActivityPane
		default:
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardAddChecklist:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.checklistInput.Value())
			if name == "" {
				m.mode = cardView
				return m, nil
			}
			m.mode = cardView
			m.pendingAction = "Creating checklist..."
			client := m.client
			cardID := m.card.ID
			return m, func() tea.Msg {
				cl, err := client.CreateChecklist(cardID, name)
				return ChecklistCreatedMsg{Checklist: cl, Err: err}
			}
		case "esc":
			m.mode = cardView
			if len(m.checklists) > 0 {
				m.mode = cardChecklistPane
			}
		default:
			var cmd tea.Cmd
			m.checklistInput, cmd = m.checklistInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardAddCheckItem:
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.checklistInput.Value())
			if name == "" {
				m.mode = cardChecklistPane
				return m, nil
			}
			refs := m.allCheckItemRefs()
			var checklistID string
			if len(refs) > 0 && m.checkItemIdx < len(refs) {
				checklistID = m.checklists[refs[m.checkItemIdx].cl].ID
			} else if len(m.checklists) > 0 {
				checklistID = m.checklists[len(m.checklists)-1].ID
			}
			m.mode = cardChecklistPane
			m.pendingAction = "Adding item..."
			client := m.client
			return m, func() tea.Msg {
				item, err := client.CreateCheckItem(checklistID, name)
				return CheckItemCreatedMsg{ChecklistID: checklistID, CheckItem: item, Err: err}
			}
		case "esc":
			m.mode = cardChecklistPane
		default:
			var cmd tea.Cmd
			m.checklistInput, cmd = m.checklistInput.Update(msg)
			return m, cmd
		}
		return m, nil

	case cardAddAttachment:
		backMode := cardView
		if len(m.attachments) > 0 {
			backMode = cardAttachmentsPane
		}
		switch msg.String() {
		case "enter":
			urlVal := strings.TrimSpace(m.checklistInput.Value())
			if urlVal == "" {
				m.mode = backMode
				return m, nil
			}
			m.mode = backMode
			m.pendingAction = "Adding attachment..."
			client := m.client
			cardID := m.card.ID
			return m, func() tea.Msg {
				att, err := client.AddAttachmentURL(cardID, urlVal)
				return AttachmentAddedMsg{Attachment: att, Err: err}
			}
		case "esc":
			m.mode = backMode
		default:
			var cmd tea.Cmd
			m.checklistInput, cmd = m.checklistInput.Update(msg)
			return m, cmd
		}
		return m, nil

	default: // cardView — info pane active
		switch msg.String() {
		case "?":
			m.showHelp = true
		case "j", "down":
			m.infoScroll++
		case "k", "up":
			if m.infoScroll > 0 {
				m.infoScroll--
			}
		case "tab":
			m.infoScroll = 0
			if len(m.checklists) > 0 {
				m.mode = cardChecklistPane
			} else if len(m.attachments) > 0 {
				m.mode = cardAttachmentsPane
			} else {
				m.mode = cardActivityPane
			}
		case "t":
			m.mode = cardEditTitle
			m.titleEdit.SetValue(m.card.Name)
			m.titleEdit.Focus()
			return m, textinput.Blink
		case "e":
			m.mode = cardEditDesc
			m.descEdit.SetValue(m.card.Desc)
			// size to fill the info pane
			available := m.height
			if available < 24 {
				available = 24
			}
			showChecklist := m.loadingCL || len(m.checklists) > 0
			showAtt := len(m.attachments) > 0
			var pane1H int
			switch {
			case showChecklist && showAtt:
				pane1H = available * infoPanePercent4 / 100
			case showChecklist || showAtt:
				pane1H = available * infoPanePercent3CL / 100
			default:
				pane1H = available * infoPanePercent2 / 100
			}
			if pane1H < minInfoPaneHeight {
				pane1H = minInfoPaneHeight
			}
			innerW := m.width - 2 - 4 // outer padding(2) + pane borders+padding(4)
			if innerW < 20 {
				innerW = 20
			}
			innerH := pane1H - 5 // borders(2) + title+blank(2) + hint(1)
			if innerH < 3 {
				innerH = 3
			}
			m.descEdit.SetWidth(innerW)
			m.descEdit.SetHeight(innerH)
			m.descEdit.Focus()
			return m, textarea.Blink
		case "m":
			m.mode = cardMoveList
			m.moveIndex = m.listIndex
		case "a":
			m.mode = cardAddMember
			m.memberIndex = 0
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			if len(m.boardMembers) == 0 {
				return m, tea.Batch(textinput.Blink, m.fetchBoardMembers())
			}
			return m, textinput.Blink
		case "l":
			m.mode = cardAddLabel
			m.labelIndex = 0
			m.pickerFilter.SetValue("")
			m.pickerFilter.Focus()
			if len(m.boardLabels) == 0 {
				return m, tea.Batch(textinput.Blink, m.fetchBoardLabels())
			}
			return m, textinput.Blink
		case "d":
			m.mode = cardSetDue
			currentVal := ""
			if m.card.Due != "" {
				for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
					if t, err := time.Parse(layout, m.card.Due); err == nil {
						currentVal = t.Format("2006-01-02")
						break
					}
				}
			}
			m.dueInput.SetValue(currentVal)
			m.dueInput.Focus()
			return m, textinput.Blink
		case "-", "N":
			m.mode = cardAddChecklist
			m.checklistInput.Placeholder = "Checklist name..."
			m.checklistInput.SetValue("")
			m.checklistInput.Focus()
			return m, textinput.Blink
		case "A":
			m.mode = cardAddAttachment
			m.checklistInput.Placeholder = "URL..."
			m.checklistInput.SetValue("")
			m.checklistInput.Focus()
			return m, textinput.Blink
		case "c":
			if m.card.ShortURL != "" {
				return m, m.copyCardURL()
			}
		case ",":
			if m.listIndex > 0 {
				m.moveIndex = m.listIndex - 1
				return m, m.moveToList(m.moveIndex)
			}
		case ".":
			if m.listIndex < len(m.lists)-1 {
				m.moveIndex = m.listIndex + 1
				return m, m.moveToList(m.moveIndex)
			}
		case "<":
			if m.listIndex > 0 {
				m.moveIndex = 0
				return m, m.moveToList(0)
			}
		case ">":
			if m.listIndex < len(m.lists)-1 {
				m.moveIndex = len(m.lists) - 1
				return m, m.moveToList(len(m.lists) - 1)
			}
		}
	}
	return m, nil
}
