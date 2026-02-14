// ====================================================================
// 電車でGO!コントローラでTSマスコンをエミュレートしてBVEで遊ぶ。
// 2021.06.01 7M4MON
// https://github.com/KeKe115/Densya_de_go
// を元に全面的に処理を見直し。
//
// TSマスコンのコマンドとBVEの内部処理の解説
// https://scrapbox.io/p4ken/入力デバイスプラグイン制作講座
//
// TinyGo版に移植
// ====================================================================
package main

import (
	"machine"
	"time"

	"gpsx"
)

// ====================================================================
// シリアルコマンドの定義
// ====================================================================

// 受ける側がブレーキとノッチを同時に入れることを想定していない。
// TSA50 はブレーキが解除されたのか、ノッチが解除されたのか判断できない。
var tsCmdHandle = [15]string{
	"TSB20", "TSB30", "TSB40", "TSE99", "TSA05", "TSA15", "TSA25", "TSA35",
	"TSA45", "TSA50", "TSA55", "TSA65", "TSA75", "TSA85", "TSA95",
}

var tsCmdButtonA = [2]string{"TSX00", "TSX99"}
var tsCmdButtonB = [2]string{"TSY00", "TSY99"}
var tsCmdButtonC = [2]string{"TSZ00", "TSZ99"}
var tsCmdButtonStart = [2]string{"TSK00", "TSK99"}
var tsCmdButtonReverser = [3]string{"TSG50", "TSG99", "TSG00"} // 中立, 前, 後
var reverserState = [4]uint8{0, 1, 0, 2}                       // セレクトボタンで状態遷移 中立→前進→中立→後退

// MasconState はマスコンの状態を表す構造体
type MasconState struct {
	notch        uint8
	brake        uint8
	handle       uint8
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

var psx *gpsx.GPSX

func main() {
	// UARTの初期化 (TSマスコンは19200bps、デバッグ用に115200bps)
	machine.Serial.Configure(machine.UARTConfig{
		BaudRate: 115200,
	})

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

	// レバーサの初期状態を送信
	println("TSG50")

	// 状態保持用変数
	var reverserPosition uint8 = 0
	lastMasconState := MasconState{
		notch:        0xff,
		brake:        0xff,
		handle:       0xff,
		buttonA:      false,
		buttonB:      false,
		buttonC:      false,
		buttonStart:  false,
		buttonSelect: false,
	}

	// メインループ
	for {
		masconState := getMasconState()

		// コントローラの状態に応じてTSマスコンのコマンドを投げる
		if lastMasconState.handle != masconState.handle && masconState.handle != 0xff {
			// 前回から変わったとき、かつ正常に取得できたときだけ
			println(tsCmdHandle[masconState.handle])
			println("---")
			lastMasconState.handle = masconState.handle
		}

		if lastMasconState.buttonA != masconState.buttonA {
			idx := boolToInt(masconState.buttonA)
			println(tsCmdButtonA[idx])
			println("---")
			lastMasconState.buttonA = masconState.buttonA
		}

		if lastMasconState.buttonB != masconState.buttonB {
			idx := boolToInt(masconState.buttonB)
			println(tsCmdButtonB[idx])
			println("---")
			lastMasconState.buttonB = masconState.buttonB
		}

		if lastMasconState.buttonC != masconState.buttonC {
			idx := boolToInt(masconState.buttonC)
			println(tsCmdButtonC[idx])
			println("---")
			lastMasconState.buttonC = masconState.buttonC
		}

		if lastMasconState.buttonStart != masconState.buttonStart {
			idx := boolToInt(masconState.buttonStart)
			println(tsCmdButtonStart[idx])
			println("---")
			lastMasconState.buttonStart = masconState.buttonStart
		}

		if lastMasconState.buttonSelect != masconState.buttonSelect {
			// セレクトは押されるたびにレバーサの状態を変える
			if masconState.buttonSelect {
				// 押されたとき
				reverserPosition++
				if reverserPosition > 3 {
					reverserPosition = 0
				}
				println(tsCmdButtonReverser[reverserState[reverserPosition]])
				println("---")
			}
			// 離されたときは処理なし
			lastMasconState.buttonSelect = masconState.buttonSelect
		}

		// ポーリング間隔が65ms以上開くとワンハンドルタイプではリセットされる。(2~60がOK範囲)
		time.Sleep(20 * time.Millisecond)
	}
}

// boolToInt converts bool to int (false=0, true=1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

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
	// 戻り値を初期化
	currentState := MasconState{
		notch:        0xff,
		brake:        0xff,
		handle:       0xff,
		buttonA:      false,
		buttonB:      false,
		buttonC:      false,
		buttonStart:  false,
		buttonSelect: false,
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

	// ハンドルポジションは正常に取得できた場合のみ更新
	if currentState.brake != 0xff && currentState.notch != 0xff {
		// ブレーキが入っていない場合のみノッチを返す。ブレーキ解除が9なのでオフセットする。
		if currentState.brake != 0 {
			currentState.handle = 9 - currentState.brake
		} else {
			currentState.handle = currentState.notch + 9
		}
	} else {
		currentState.handle = 0xff
	}

	// ボタン状態の取得
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonSquare) {
		currentState.buttonA = true
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonCross) {
		currentState.buttonB = true
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonCircle) {
		currentState.buttonC = true
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonStart) {
		currentState.buttonStart = true
	}
	if psx.IsDown(gpsx.Pad1, gpsx.ButtonSelect) {
		currentState.buttonSelect = true
	}

	return currentState
}
