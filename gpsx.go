// Package gpsx provides a TinyGo driver for PS2 controllers.
// This is a port of the Arduino GPSX library.
package gpsx

import (
	"machine"
	"time"
)

// Platform type constants
const (
	PS1 uint8 = 0
	PS2 uint8 = 1
)

// Pad selection constants
const (
	Pad1 uint8 = 0
	Pad2 uint8 = 1
)

// Buffer size for controller state
const PadBufferSize = 20

// Motor control constants
const (
	Motor1Enable  uint8 = 0x00
	Motor1Disable uint8 = 0xFF
	Motor2Enable  uint8 = 0x01
	Motor2Disable uint8 = 0xFF
	Motor1On      uint8 = 0xFF
	Motor1Off     uint8 = 0x00
)

// Mode constants
const (
	ModeDigital uint8 = 0x00
	ModeAnalog  uint8 = 0x01
	ModeLock    uint8 = 0x03
	ModeUnlock  uint8 = 0x02
)

// State indices
const (
	stateCurrent  = 0
	statePrevious = 1
)

// PinConfig holds the pin configuration for the PS2 controller interface.
type PinConfig struct {
	DAT machine.Pin // Data input (requires pull-up)
	CMD machine.Pin // Command output
	CLK machine.Pin // Clock output
	AT1 machine.Pin // Attention for PAD1
	AT2 machine.Pin // Attention for PAD2 (optional if using only PAD1)
	ACK machine.Pin // ACK input (not used but defined)
}

// GPSX is the main controller interface.
type GPSX struct {
	psxType uint8
	pins    PinConfig

	// Timing configuration
	waitAfterATT    time.Duration
	commandInterval time.Duration
	clkHalfCycle    time.Duration
	ackWaitDuration time.Duration

	// State management (2 pads x 2 states x buffer size)
	keyState [2][2][PadBufferSize]byte
	padState [PadBufferSize]byte

	// Motor levels for each pad
	motor1Level [2]uint8
	motor2Level [2]uint8
}

// New creates a new GPSX controller with the specified platform type and pins.
func New(psxType uint8, pins PinConfig) *GPSX {
	g := &GPSX{
		psxType: psxType,
		pins:    pins,
	}

	// Set timing based on platform
	if psxType == PS1 {
		g.waitAfterATT = 50 * time.Microsecond
		g.commandInterval = 16 * time.Millisecond
		g.clkHalfCycle = 2 * time.Microsecond
		g.ackWaitDuration = 15 * time.Microsecond
	} else { // PS2
		g.waitAfterATT = 15 * time.Microsecond
		g.commandInterval = 10 * time.Millisecond
		g.clkHalfCycle = 1 * time.Microsecond
		g.ackWaitDuration = 15 * time.Microsecond
	}

	// Initialize motor levels
	g.motor1Level = [2]uint8{Motor1Off, Motor1Off}
	g.motor2Level = [2]uint8{0x00, 0x00}

	g.init()
	return g
}

// init configures the GPIO pins and sets their initial states.
func (g *GPSX) init() {
	// Configure pins
	g.pins.CLK.Configure(machine.PinConfig{Mode: machine.PinOutput})
	g.pins.CMD.Configure(machine.PinConfig{Mode: machine.PinOutput})
	g.pins.DAT.Configure(machine.PinConfig{Mode: machine.PinInput})
	g.pins.AT1.Configure(machine.PinConfig{Mode: machine.PinOutput})

	// Configure AT2 only if it's set (non-zero)
	if g.pins.AT2 != 0 {
		g.pins.AT2.Configure(machine.PinConfig{Mode: machine.PinOutput})
	}

	// Configure ACK only if it's set (non-zero)
	if g.pins.ACK != 0 {
		g.pins.ACK.Configure(machine.PinConfig{Mode: machine.PinInput})
	}

	// Set initial pin states
	g.pins.CLK.High()
	g.pins.CMD.Low()
	g.pins.AT1.High()
	if g.pins.AT2 != 0 {
		g.pins.AT2.High()
	}
}

// UpdateState polls the controller and updates button states.
func (g *GPSX) UpdateState(pad uint8) {
	// Swap current and previous states (copy in Go)
	g.keyState[pad][statePrevious] = g.keyState[pad][stateCurrent]

	// Prepare poll command with motor values
	pollCmd := []byte{0x01, 0x42, 0x00, g.motor1Level[pad], g.motor2Level[pad]}

	// Send poll command
	g.sendCommand(pad, pollCmd)

	// Copy response to current state
	for i := 0; i < 9; i++ {
		g.keyState[pad][stateCurrent][i] = g.padState[i]
	}

	// For digital keys, previous state is stored as a mask (XOR)
	// of bits changed from previous poll.
	g.keyState[pad][statePrevious][3] ^= g.keyState[pad][stateCurrent][3]
	g.keyState[pad][statePrevious][4] ^= g.keyState[pad][stateCurrent][4]
}

// Motor sets the motor levels (takes effect on next UpdateState).
func (g *GPSX) Motor(pad uint8, motor1OnOff uint8, motor2Level uint8) {
	g.motor1Level[pad] = motor1OnOff
	g.motor2Level[pad] = motor2Level
}

// MotorEnable enables or disables motors on the controller.
func (g *GPSX) MotorEnable(pad uint8, motor1Enable uint8, motor2Enable uint8) {
	cmdEnableMotor := []byte{0x01, 0x4D, 0x00, motor1Enable, motor2Enable, 0xFF, 0xFF, 0xFF, 0xFF}
	cmdEnterCfg := []byte{0x01, 0x43, 0x00, 0x01}
	cmdExitCfg := []byte{0x01, 0x43, 0x00, 0x00, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A}

	g.sendCommand(pad, cmdEnterCfg)
	g.sendCommand(pad, cmdEnableMotor)
	g.sendCommand(pad, cmdExitCfg)
}

// Mode sets the analog/digital mode and lock state.
func (g *GPSX) Mode(pad uint8, mode uint8, lock uint8) {
	cmdADMode := []byte{0x01, 0x44, 0x00, mode, lock, 0x00, 0x00, 0x00, 0x00}
	cmdEnterCfg := []byte{0x01, 0x43, 0x00, 0x01}
	cmdExitCfg := []byte{0x01, 0x43, 0x00, 0x00, 0x5A, 0x5A, 0x5A, 0x5A, 0x5A}

	g.sendCommand(pad, cmdEnterCfg)
	g.sendCommand(pad, cmdADMode)
	g.sendCommand(pad, cmdExitCfg)
}
