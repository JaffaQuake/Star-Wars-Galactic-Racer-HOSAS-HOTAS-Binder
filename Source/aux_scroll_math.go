package main

// Scrollbar codes shared by the Additional Inputs section (Win32 SB_*).
const (
	scrollLineUp = iota
	scrollLineDown
	scrollPageUp
	scrollPageDown
	scrollThumbPosition
	scrollThumbTrack
	scrollTop
	scrollBottom
)

// clampScroll keeps a scroll offset between the top and the point where the
// last row reaches the bottom edge.
func clampScroll(pos, content, view int) int {
	limit := content - view
	if pos > limit {
		pos = limit
	}
	if pos < 0 {
		pos = 0
	}
	return pos
}

// auxScrollAfter is the Additional Inputs scroll offset after a scrollbar
// action. Arrows move one input row, the track moves one visible page, and
// dragging the thumb follows it exactly.
func auxScrollAfter(code, thumb, pos, content, view, row int) int {
	switch code {
	case scrollLineUp:
		pos -= row
	case scrollLineDown:
		pos += row
	case scrollPageUp:
		pos -= max(row, view/row*row)
	case scrollPageDown:
		pos += max(row, view/row*row)
	case scrollThumbPosition, scrollThumbTrack:
		pos = thumb
	case scrollTop:
		pos = 0
	case scrollBottom:
		pos = content
	}
	return clampScroll(pos, content, view)
}

// auxWheelScroll scrolls one input row per wheel notch (positive delta is
// away from the user, so up). moved is false at either end, so the caller
// can pass the wheel on to the main window.
func auxWheelScroll(delta, pos, content, view, row int) (next int, moved bool) {
	steps := delta / 120
	if steps == 0 && delta != 0 {
		steps = 1
		if delta < 0 {
			steps = -1
		}
	}
	next = clampScroll(pos-steps*row, content, view)
	return next, next != pos
}
