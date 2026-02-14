// Package gpsx provides a TinyGo driver for PlayStation 2 controllers.
//
// This package is a port of the Arduino GPSX library, implementing
// bit-banging SPI communication to interface with PS2/PS1 controllers.
//
// # Features
//
//   - Support for both PS1 and PS2 controllers
//   - Digital and analog mode support
//   - Dual controller support (PAD1 and PAD2)
//   - Motor/vibration control
//   - Edge detection for button press/release events
//
// # Hardware Connection
//
// The PS2 controller uses a custom serial protocol. Connect the controller
// as follows (1kΩ pull-up resistors required on DAT, AT1, AT2 pins):
//
//	Controller Pin | Function     | Notes
//	---------------|--------------|---------------------------
//	1              | DAT (Data)   | Input, requires pull-up
//	2              | CMD          | Output, command to controller
//	3              | 9V Motor     | Optional, for vibration
//	4              | GND          | Ground
//	5              | VCC          | 3.3V power
//	6              | ATT          | Attention, active low
//	7              | CLK          | Clock, output
//	8              | N/C          | Not connected
//	9              | ACK          | Acknowledge (optional)
//
// # Example Usage
//
//	package main
//
//	import (
//	    "machine"
//	    "time"
//	    "path/to/gpsx"
//	)
//
//	func main() {
//	    pins := gpsx.PinConfig{
//	        DAT: machine.GP2,
//	        CMD: machine.GP3,
//	        CLK: machine.GP4,
//	        AT1: machine.GP5,
//	    }
//
//	    psx := gpsx.New(gpsx.PS2, pins)
//	    psx.Mode(gpsx.Pad1, gpsx.ModeAnalog, gpsx.ModeLock)
//
//	    for {
//	        psx.UpdateState(gpsx.Pad1)
//
//	        if psx.Pressed(gpsx.Pad1, gpsx.ButtonCircle) {
//	            println("Circle pressed!")
//	        }
//
//	        if psx.IsDown(gpsx.Pad1, gpsx.ButtonCross) {
//	            println("Cross is held down")
//	        }
//
//	        // Read analog sticks (only in analog mode)
//	        x := psx.AnalogLeftX(gpsx.Pad1)
//	        y := psx.AnalogLeftY(gpsx.Pad1)
//	        println("Left stick:", x, y)
//
//	        time.Sleep(20 * time.Millisecond)
//	    }
//	}
//
// # Original Library
//
// This is a TinyGo port of the Arduino GPSX library by Studio Gyokimae.
// Original documentation: https://pspunch.com/pd/article/arduino_lib_gpsx.html
package gpsx
