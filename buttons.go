package gpsx

// Button represents a button on the PS2 controller.
type Button struct {
	byteIndex uint8
	bitMask   uint8
}

// Button definitions
var (
	// D-Pad and special buttons (byte index 3)
	ButtonLeft       = Button{3, 1 << 7}
	ButtonDown       = Button{3, 1 << 6}
	ButtonRight      = Button{3, 1 << 5}
	ButtonUp         = Button{3, 1 << 4}
	ButtonStart      = Button{3, 1 << 3}
	ButtonStickRight = Button{3, 1 << 2} // R3
	ButtonStickLeft  = Button{3, 1 << 1} // L3
	ButtonSelect     = Button{3, 1 << 0}

	// Action buttons and shoulder buttons (byte index 4)
	ButtonSquare   = Button{4, 1 << 7}
	ButtonCross    = Button{4, 1 << 6}
	ButtonCircle   = Button{4, 1 << 5}
	ButtonTriangle = Button{4, 1 << 4}
	ButtonR1       = Button{4, 1 << 3}
	ButtonL1       = Button{4, 1 << 2}
	ButtonR2       = Button{4, 1 << 1}
	ButtonL2       = Button{4, 1 << 0}
)

// IsDown returns true if the button is currently pressed.
// Note: Buttons are active LOW (0 = pressed, 1 = released).
func (g *GPSX) IsDown(pad uint8, btn Button) bool {
	return g.keyState[pad][stateCurrent][btn.byteIndex]&btn.bitMask == 0
}

// Pressed returns true if the button was just pressed (transition from up to down).
// This uses edge detection and only returns true once per button press.
func (g *GPSX) Pressed(pad uint8, btn Button) bool {
	prev := g.keyState[pad][statePrevious][btn.byteIndex] & btn.bitMask
	curr := g.keyState[pad][stateCurrent][btn.byteIndex] & btn.bitMask
	// prev contains XOR of previous and current, so if bit is set, state changed
	// curr == 0 means button is now pressed
	return prev != 0 && curr == 0
}

// Released returns true if the button was just released (transition from down to up).
// This uses edge detection and only returns true once per button release.
func (g *GPSX) Released(pad uint8, btn Button) bool {
	prev := g.keyState[pad][statePrevious][btn.byteIndex] & btn.bitMask
	curr := g.keyState[pad][stateCurrent][btn.byteIndex] & btn.bitMask
	// prev contains XOR of previous and current, so if bit is set, state changed
	// curr != 0 means button is now released
	return prev != 0 && curr != 0
}

// AnalogRightX returns the right analog stick X-axis value (0-255).
// Only valid in analog mode.
func (g *GPSX) AnalogRightX(pad uint8) uint8 {
	return g.keyState[pad][stateCurrent][5]
}

// AnalogRightY returns the right analog stick Y-axis value (0-255).
// Only valid in analog mode.
func (g *GPSX) AnalogRightY(pad uint8) uint8 {
	return g.keyState[pad][stateCurrent][6]
}

// AnalogLeftX returns the left analog stick X-axis value (0-255).
// Only valid in analog mode.
func (g *GPSX) AnalogLeftX(pad uint8) uint8 {
	return g.keyState[pad][stateCurrent][7]
}

// AnalogLeftY returns the left analog stick Y-axis value (0-255).
// Only valid in analog mode.
func (g *GPSX) AnalogLeftY(pad uint8) uint8 {
	return g.keyState[pad][stateCurrent][8]
}

// IsAnalog returns true if the controller is in analog mode.
func (g *GPSX) IsAnalog(pad uint8) bool {
	return g.keyState[pad][stateCurrent][1]&0xF0 == 0x70
}

// IsDigital returns true if the controller is in digital mode.
func (g *GPSX) IsDigital(pad uint8) bool {
	return g.keyState[pad][stateCurrent][1]&0xF0 == 0x40
}
