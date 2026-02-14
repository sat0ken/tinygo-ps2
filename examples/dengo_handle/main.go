// ====================================================================
// 電車でGO!コントローラでBVEをキーボード操作する
//
// USB HIDキーボードとしてBVEの標準キー操作をエミュレートする。
// dengo_ts_emuのTSマスコンプロトコル版と異なり、プラグイン不要で
// BVEを操作可能。
//
// BVE キー割り当て: https://bvets.net/old/drive/key.html
//   Z: マスコン+, A: マスコン-, .: ブレーキ+, ,: ブレーキ-
//   Space: ATS確認, Enter: 警笛, Delete: EB解除
//   BackSpace: 停止, 上下矢印: レバーサー
//
// 参考: https://github.com/sago35/tinygo-workshop-keyboard/blob/main/04_usbhid_keyboard/main.go
// ====================================================================
package main

import (
	"machine"
	"machine/usb/hid/keyboard"
	"time"

	"gpsx"
)

// ====================================================================
// マスコン状態
// ====================================================================

// MasconState はマスコンの状態を表す構造体
type MasconState struct {
	notch        uint8
	brake        uint8
	buttonA      bool
	buttonB      bool
	buttonC      bool
	buttonStart  bool
	buttonSelect bool
}

// ハンドル接点の定義
const (
	handleContact1 uint8 = 0b0001
	handleContact2 uint8 = 0b0010
	handleContact3 uint8 = 0b0100
	handleContact4 uint8 = 0b1000
)

// ノッチポジションのマッピング
const (
	mapNotchOff uint8 = 0b0111
	mapNotch1   uint8 = 0b1110
	mapNotch2   uint8 = 0b0110
	mapNotch3   uint8 = 0b1011
	mapNotch4   uint8 = 0b0011
	mapNotch5   uint8 = 0b1010
)

// ブレーキポジションのマッピング
const (
	mapBrakeOff  uint8 = 0b1101
	mapBrake1    uint8 = 0b0111
	mapBrake2    uint8 = 0b0101
	mapBrake3    uint8 = 0b1110
	mapBrake4    uint8 = 0b1100
	mapBrake5    uint8 = 0b0110
	mapBrake6    uint8 = 0b0100
	mapBrake7    uint8 = 0b1011
	mapBrake8    uint8 = 0b1001
	mapBrakeEmer uint8 = 0b0000
)

// レバーサーの状態遷移: 中立→前進→中立→後退
var reverserState = [4]uint8{0, 1, 0, 2} // 0:中立, 1:前進, 2:後退

var psx *gpsx.GPSX

func main() {
	// USB HIDキーボードの初期化
	kb := keyboard.Port()

	// PSコントローラのピン設定（使用するボードに合わせて変更してください）
	pins := gpsx.PinConfig{
		DAT: machine.D26,
		CMD: machine.D15,
		CLK: machine.D27,
		AT1: machine.D14,
	}

	// PSコントローラライブラリの初期化
	psx = gpsx.New(gpsx.PS2, pins)
	psx.Mode(gpsx.Pad1, gpsx.ModeDigital, gpsx.ModeLock)
	psx.MotorEnable(gpsx.Pad1, gpsx.Motor1Disable, gpsx.Motor2Disable)

	// 状態保持用変数
	var lastNotch uint8 = 0
	var lastBrake uint8 = 0
	var lastButtonA bool
	var lastButtonB bool
	var lastButtonC bool
	var lastButtonStart bool
	var lastButtonSelect bool
	var reverserPosition uint8 = 0
	var lastReverserValue uint8 = 0 // 0:中立, 1:前進, 2:後退

	// USB初期化待ち
	time.Sleep(2 * time.Second)

	// メインループ
	for {
		state := getMasconState()

		// ====================================================
		// ノッチの差分 → Z（マスコン+）/ A（マスコン-）
		// ====================================================
		if state.notch != 0xff && state.notch != lastNotch {
			diff := int(state.notch) - int(lastNotch)
			if diff > 0 {
				for i := 0; i < diff; i++ {
					kb.Press(keyboard.KeyZ)
				}
			} else {
				for i := 0; i < -diff; i++ {
					kb.Press(keyboard.KeyA)
				}
			}
			lastNotch = state.notch
		}

		// ====================================================
		// ブレーキの差分 → .（ブレーキ+）/ ,（ブレーキ-）
		// ====================================================
		if state.brake != 0xff && state.brake != lastBrake {
			diff := int(state.brake) - int(lastBrake)
			if diff > 0 {
				for i := 0; i < diff; i++ {
					kb.Press(keyboard.KeyPeriod)
				}
			} else {
				for i := 0; i < -diff; i++ {
					kb.Press(keyboard.KeyComma)
				}
			}
			lastBrake = state.brake
		}

		// ====================================================
		// □ (Square) → Space（ATS確認）
		// ====================================================
		if lastButtonA != state.buttonA {
			if state.buttonA {
				kb.Down(keyboard.KeySpace)
			} else {
				kb.Up(keyboard.KeySpace)
			}
			lastButtonA = state.buttonA
		}

		// ====================================================
		// × (Cross) → Enter（警笛）
		// ====================================================
		if lastButtonB != state.buttonB {
			if state.buttonB {
				kb.Down(keyboard.KeyEnter)
			} else {
				kb.Up(keyboard.KeyEnter)
			}
			lastButtonB = state.buttonB
		}

		// ====================================================
		// ○ (Circle) → Delete（EB解除）
		// ====================================================
		if lastButtonC != state.buttonC {
			if state.buttonC {
				kb.Down(keyboard.KeyDelete)
			} else {
				kb.Up(keyboard.KeyDelete)
			}
			lastButtonC = state.buttonC
		}

		// ====================================================
		// START → Backspace（停止）
		// ====================================================
		if lastButtonStart != state.buttonStart {
			if state.buttonStart {
				kb.Down(keyboard.KeyBackspace)
			} else {
				kb.Up(keyboard.KeyBackspace)
			}
			lastButtonStart = state.buttonStart
		}

		// ====================================================
		// SELECT → レバーサー（上下矢印キーで遷移）
		// 状態遷移: 中立(0)→前進(1)→中立(0)→後退(2)→...
		// ====================================================
		if lastButtonSelect != state.buttonSelect {
			if state.buttonSelect {
				// 押されたとき: レバーサーの状態を遷移
				reverserPosition++
				if reverserPosition > 3 {
					reverserPosition = 0
				}
				newReverserValue := reverserState[reverserPosition]

				// 状態変化に応じて上下矢印キーを送信
				switch {
				case lastReverserValue == 0 && newReverserValue == 1: // 中立→前進
					kb.Press(keyboard.KeyUp)
				case lastReverserValue == 1 && newReverserValue == 0: // 前進→中立
					kb.Press(keyboard.KeyDown)
				case lastReverserValue == 0 && newReverserValue == 2: // 中立→後退
					kb.Press(keyboard.KeyDown)
				case lastReverserValue == 2 && newReverserValue == 0: // 後退→中立
					kb.Press(keyboard.KeyUp)
				}
				lastReverserValue = newReverserValue
			}
			lastButtonSelect = state.buttonSelect
		}

		// ポーリング間隔（2~60msがOK範囲）
		time.Sleep(20 * time.Millisecond)
	}
}

// ====================================================================
// マスコン状態の取得関数
// ====================================================================

// getNotchState はノッチポジションから状態を取得する
func getNotchState(notchPosition uint8) uint8 {
	switch notchPosition {
	case mapNotchOff:
		return 0
	case mapNotch1:
		return 1
	case mapNotch2:
		return 2
	case mapNotch3:
		return 3
	case mapNotch4:
		return 4
	case mapNotch5:
		return 5
	default:
		return 0xff // 中途半端な状態は取得しない
	}
}

// getBrakeState はブレーキポジションから状態を取得する
func getBrakeState(brakePosition uint8) uint8 {
	switch brakePosition {
	case mapBrakeOff:
		return 0
	case mapBrake1:
		return 1
	case mapBrake2:
		return 2
	case mapBrake3:
		return 3
	case mapBrake4:
		return 4
	case mapBrake5:
		return 5
	case mapBrake6:
		return 6
	case mapBrake7:
		return 7
	case mapBrake8:
		return 8
	case mapBrakeEmer:
		return 9
	default:
		return 0xff // 中途半端な状態は取得しない
	}
}

// getMasconState はマスコンの状態を取得する
func getMasconState() MasconState {
	currentState := MasconState{
		notch: 0xff,
		brake: 0xff,
	}

	// PSコントローラの状態更新
	psx.UpdateState(gpsx.Pad1)

	// ノッチ状態の取得
	var notchPosition uint8 = 0
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonLeft) {
		notchPosition |= handleContact1
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonDown) {
		notchPosition |= handleContact2
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonRight) {
		notchPosition |= handleContact3
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonTriangle) {
		notchPosition |= handleContact4
	}
	currentState.notch = getNotchState(notchPosition)

	// ブレーキ状態の取得
	var brakePosition uint8 = 0
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonR1) {
		brakePosition |= handleContact1
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonL1) {
		brakePosition |= handleContact2
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonR2) {
		brakePosition |= handleContact3
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonL2) {
		brakePosition |= handleContact4
	}
	currentState.brake = getBrakeState(brakePosition)

	// ボタン状態の取得
	currentState.buttonA = psx.IsDown(gpsx.Pad1, gpsx.ButtonSquare)
	currentState.buttonB = psx.IsDown(gpsx.Pad1, gpsx.ButtonCross)
	currentState.buttonC = psx.IsDown(gpsx.Pad1, gpsx.ButtonCircle)
	currentState.buttonStart = psx.IsDown(gpsx.Pad1, gpsx.ButtonStart)
	currentState.buttonSelect = psx.IsDown(gpsx.Pad1, gpsx.ButtonSelect)

	return currentState
}
