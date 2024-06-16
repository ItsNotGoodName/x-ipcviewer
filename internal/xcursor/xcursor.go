// xcursor forked from https://github.com/BurntSushi/xgbutil/blob/master/xcursor/xcursor.go
package xcursor

import (
	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	XCursor           = 0
	Arrow             = 2
	BasedArrowDown    = 4
	BasedArrowUp      = 6
	Boat              = 8
	Bogosity          = 10
	BottomLeftCorner  = 12
	BottomRightCorner = 14
	BottomSide        = 16
	BottomTee         = 18
	BoxSpiral         = 20
	CenterPtr         = 22
	Circle            = 24
	Clock             = 26
	CoffeeMug         = 28
	Cross             = 30
	CrossReverse      = 32
	Crosshair         = 34
	DiamondCross      = 36
	Dot               = 38
	DotBoxMask        = 40
	DoubleArrow       = 42
	DraftLarge        = 44
	DraftSmall        = 46
	DrapedBox         = 48
	Exchange          = 50
	Fleur             = 52
	Gobbler           = 54
	Gumby             = 56
	Hand1             = 58
	Hand2             = 60
	Heart             = 62
	Icon              = 64
	IronCross         = 66
	LeftPtr           = 68
	LeftSide          = 70
	LeftTee           = 72
	LeftButton        = 74
	LLAngle           = 76
	LRAngle           = 78
	Man               = 80
	MiddleButton      = 82
	Mouse             = 84
	Pencil            = 86
	Pirate            = 88
	Plus              = 90
	QuestionArrow     = 92
	RightPtr          = 94
	RightSide         = 96
	RightTee          = 98
	RightButton       = 100
	RtlLogo           = 102
	Sailboat          = 104
	SBDownArrow       = 106
	SBHDoubleArrow    = 108
	SBLeftArrow       = 110
	SBRightArrow      = 112
	SBUpArrow         = 114
	SBVDoubleArrow    = 116
	Shuttle           = 118
	Sizing            = 120
	Spider            = 122
	Spraycan          = 124
	Star              = 126
	Target            = 128
	TCross            = 130
	TopLeftArrow      = 132
	TopLeftCorner     = 134
	TopRightCorner    = 136
	TopSide           = 138
	TopTee            = 140
	Trek              = 142
	ULAngle           = 144
	Umbrella          = 146
	URAngle           = 148
	Watch             = 150
	XTerm             = 152
)

func CreateCursor(x *xgb.Conn, cursor uint16) (xproto.Cursor, error) {
	return CreateCursorExtra(x, cursor, 0xffff, 0xffff, 0xffff, 0, 0, 0)
}

func CreateCursorExtra(x *xgb.Conn, cursor, foreRed, foreGreen,
	foreBlue, backRed, backGreen, backBlue uint16) (xproto.Cursor, error) {

	fontId, err := xproto.NewFontId(x)
	if err != nil {
		return 0, err
	}

	cursorId, err := xproto.NewCursorId(x)
	if err != nil {
		return 0, err
	}

	err = xproto.OpenFontChecked(x, fontId,
		uint16(len("cursor")), "cursor").Check()
	if err != nil {
		return 0, err
	}

	err = xproto.CreateGlyphCursorChecked(x, cursorId, fontId, fontId,
		cursor, cursor+1,
		foreRed, foreGreen, foreBlue,
		backRed, backGreen, backBlue).Check()
	if err != nil {
		return 0, err
	}

	err = xproto.CloseFontChecked(x, fontId).Check()
	if err != nil {
		return 0, err
	}

	return cursorId, nil
}
