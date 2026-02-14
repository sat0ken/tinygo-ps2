// Example program demonstrating GPSX library usage with TinyGo.
// Target: Raspberry Pi Pico
package main

import (
	"machine"
	"time"

	"gpsx"
)

func main() {
	// Configure pins for Raspberry Pi Pico
	pins := gpsx.PinConfig{
		DAT: machine.GP2, // Data input (requires external 1k pull-up)
		CMD: machine.GP3, // Command output
		CLK: machine.GP4, // Clock output
		AT1: machine.GP5, // Attention for PAD1
		AT2: machine.GP6, // Attention for PAD2 (optional)
	}

	// Initialize PS2 controller interface
	psx := gpsx.New(gpsx.PS2, pins)

	// Set analog mode with lock (prevent mode change by analog button)
	psx.Mode(gpsx.Pad1, gpsx.ModeAnalog, gpsx.ModeLock)

	// Enable motors
	psx.MotorEnable(gpsx.Pad1, gpsx.Motor1Enable, gpsx.Motor2Enable)

	// Initial state poll
	psx.UpdateState(gpsx.Pad1)

	for {
		// Poll controller state
		psx.UpdateState(gpsx.Pad1)

		// Check for button presses (edge detection)
		if psx.Pressed(gpsx.Pad1, gpsx.ButtonCircle) {
			println("Circle pressed!")
			// Turn on motor 1
			psx.Motor(gpsx.Pad1, gpsx.Motor1On, 0x00)
		}

		if psx.Released(gpsx.Pad1, gpsx.ButtonCircle) {
			println("Circle released!")
			// Turn off motor 1
			psx.Motor(gpsx.Pad1, gpsx.Motor1Off, 0x00)
		}

		// Check if buttons are held down
		if psx.IsDown(gpsx.Pad1, gpsx.ButtonCross) {
			println("Cross is held down")
		}

		// D-Pad input
		if psx.IsDown(gpsx.Pad1, gpsx.ButtonUp) {
			println("Up")
		}
		if psx.IsDown(gpsx.Pad1, gpsx.ButtonDown) {
			println("Down")
		}
		if psx.IsDown(gpsx.Pad1, gpsx.ButtonLeft) {
			println("Left")
		}
		if psx.IsDown(gpsx.Pad1, gpsx.ButtonRight) {
			println("Right")
		}

		// Read analog sticks (only valid in analog mode)
		if psx.IsAnalog(gpsx.Pad1) {
			lx := psx.AnalogLeftX(gpsx.Pad1)
			ly := psx.AnalogLeftY(gpsx.Pad1)
			rx := psx.AnalogRightX(gpsx.Pad1)
			ry := psx.AnalogRightY(gpsx.Pad1)

			// Print if stick is moved significantly from center (128)
			if lx < 100 || lx > 156 || ly < 100 || ly > 156 {
				println("Left stick:", lx, ly)
			}
			if rx < 100 || rx > 156 || ry < 100 || ry > 156 {
				println("Right stick:", rx, ry)
			}

			// Use left stick X to control motor 2 speed with Triangle button
			if psx.IsDown(gpsx.Pad1, gpsx.ButtonTriangle) {
				psx.Motor(gpsx.Pad1, gpsx.Motor1Off, lx)
			}
		}

		// Recommended polling interval: 10-60ms
		time.Sleep(20 * time.Millisecond)
	}
}
