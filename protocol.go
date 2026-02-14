package gpsx

import "time"

// readWriteByte performs bit-banging SPI transfer of one byte.
// This implements the PS2 controller communication protocol.
func (g *GPSX) readWriteByte(cmdByte byte) byte {
	var received byte = 0

	for i := 0; i < 8; i++ {
		// Set CMD pin based on LSB
		if cmdByte&0x01 != 0 {
			g.pins.CMD.High()
		} else {
			g.pins.CMD.Low()
		}
		cmdByte >>= 1

		// Clock down
		g.pins.CLK.Low()
		time.Sleep(g.clkHalfCycle)

		// Clock up
		g.pins.CLK.High()
		time.Sleep(g.clkHalfCycle)

		// Read DAT pin on rising edge (MSB-first reception)
		received >>= 1
		if g.pins.DAT.Get() {
			received |= 0x80
		}
	}

	return received
}

// sendCommand sends a command sequence to the specified pad and stores the response.
func (g *GPSX) sendCommand(pad uint8, msg []byte) {
	// Select attention pin based on pad number
	attPin := g.pins.AT1
	if pad == Pad2 && g.pins.AT2 != 0 {
		attPin = g.pins.AT2
	}

	// Get attention (pull low)
	attPin.Low()
	time.Sleep(g.waitAfterATT)

	// Send/receive 9 bytes
	for i := 0; i < 9; i++ {
		cmd := byte(0x00)
		if i < len(msg) {
			cmd = msg[i]
		}
		g.padState[i] = g.readWriteByte(cmd)
		time.Sleep(g.ackWaitDuration)
	}

	// Release attention (pull high)
	attPin.High()
	time.Sleep(g.commandInterval)
}
